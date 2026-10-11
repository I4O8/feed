package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// 点播模块：授权闸门（本模块核心）
// 文档：docs/点播模块.md 五、授权闸门
//
// ②③④ 的所有安全策略都收敛到同一个执行点：播放凭证闸门 + 密钥分发端点。
// ticket 用「不透明随机串 + Redis 存储」而非 JWT —— 因为本模块要「可即时吊销」
// （发现盗号要马上踢人），无状态在此是负担（JWT vs Session 的经典权衡）。

// 点播模块错误
var (
	ErrVodMovieNotFound  = errors.New("movie not found")            // 40410
	ErrVodNotReady       = errors.New("movie not ready")            // 40411
	ErrVodConcurrent     = errors.New("too many concurrent sessions") // 40301
	ErrVodTicketInvalid  = errors.New("ticket invalid or expired")  // 40110
	ErrVodDeviceMismatch = errors.New("device mismatch")            // 40302
	ErrVodVariantForbidden = errors.New("variant not allowed")      // 40303
	ErrVodSigInvalid     = errors.New("segment signature invalid")  // 40310
)

// Redis 键前缀
const (
	vodTicketPrefix = "vod:ticket:"  // 播放凭证
	vodSessPrefix   = "vod:sess:"    // 活跃会话 ZSET（member=sessionId, score=心跳时间戳）
	vodHLSKeyPrefix = "vod:hls:key:" // HLS per-session 密钥
	vodSessWMPrefix = "vod:sesswm:"  // 会话水印码（分片变体白名单 O(1) 校验用）
)

// vodTicketInfo ticket 携带的会话状态（存 Redis 的值）
type vodTicketInfo struct {
	UID         int64  `json:"uid"`
	MovieID     int64  `json:"movieId"`
	SessionID   int64  `json:"sessionId"`
	RenditionID int64  `json:"renditionId"`
	DeviceID    string `json:"deviceId"`
	WmCode      uint64 `json:"wmCode"` // 水印码：位 i = 第 i 段取 A(0)/B(1)
	Exp         int64  `json:"exp"`    // 过期 Unix 秒
}

// VodPlayResult 开播响应
type VodPlayResult struct {
	Ticket      string `json:"ticket"`
	ManifestURL string `json:"manifestUrl"`
	DRMScheme   string `json:"drmScheme"`
	LicenseURL  string `json:"licenseUrl"` // CLEARKEY 时给 license；AES128 时给 key 端点
}

// randomToken 生成不透明随机串（32 字节 → 64 位 hex）
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// genWmCode 生成水印码：随机 bits 位（段数即码长）
func genWmCode(bits int) (uint64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	code := binary.BigEndian.Uint64(b[:])
	if bits > 0 && bits < 64 {
		code &= (uint64(1) << uint(bits)) - 1
	}
	return code, nil
}

// pickRendition 按客户端偏好选产物；偏好不可用时回退到另一条线
func pickRendition(movieID int64, drmScheme string) (*model.MovieRendition, error) {
	if drmScheme == model.DRMSchemeAES128 {
		if r, _ := dao.FindRendition(movieID, model.ContainerHLS, model.DRMSchemeAES128); r != nil {
			return r, nil
		}
		return dao.FindRendition(movieID, model.ContainerDASH, model.DRMSchemeClearKey)
	}
	// 默认 CLEARKEY / DASH 主线
	if r, _ := dao.FindRendition(movieID, model.ContainerDASH, model.DRMSchemeClearKey); r != nil {
		return r, nil
	}
	return dao.FindRendition(movieID, model.ContainerHLS, model.DRMSchemeAES128)
}

// VodPlay 开播：①验登录(外层中间件) ②清过期会话+判并发 ③建会话 ④发 ticket ⑤分配水印码
func VodPlay(uid, movieID int64, deviceID, drmScheme, ip, ua string) (*VodPlayResult, error) {
	m, err := dao.FindMovieByID(movieID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrVodMovieNotFound
	}
	if m.Status != model.MovieStatusReady {
		return nil, ErrVodNotReady
	}

	rend, err := pickRendition(movieID, drmScheme)
	if err != nil {
		return nil, err
	}
	if rend == nil {
		return nil, ErrVodNotReady // 没有可用的产物 = 未就绪
	}

	// ③ 建会话（先落库拿真实 sessionID，再用它做并发判定；失败则回滚该行）
	nowTime := time.Now()
	sess := &model.PlaybackSession{
		UID:           uid,
		MovieID:       movieID,
		RenditionID:   rend.ID,
		DeviceID:      deviceID,
		IP:            ip,
		UserAgent:     ua,
		Status:        model.SessionActive,
		HeartbeatAt:   nowTime,
	}
	sessionID, err := dao.CreateSession(sess)
	if err != nil {
		return nil, err
	}

	// ⑤ 分配水印码（码长 = 段数）
	segCount := m.SegmentCount
	if segCount < 1 {
		segCount = 1
	}
	wmCode, err := genWmCode(segCount)
	if err != nil {
		return nil, err
	}
	sess.WatermarkCode = wmCode
	if err := dao.SetSessionWatermark(sessionID, wmCode); err != nil {
		return nil, err
	}
	// 水印码同步进 Redis：分片请求要高频校验变体白名单，不宜每次查库
	dao.RDB.Set(context.Background(), vodSessWMPrefix+strconv.FormatInt(sessionID, 10),
		strconv.FormatUint(wmCode, 10), time.Duration(config.VodTicketGrace+m.Duration)*time.Second)

	// ② 并发判定（原子 Lua：清死 + 判数 + 加入）；超限则把刚建的行标为结束，不留脏数据
	now := time.Now().Unix()
	acquired, err := dao.RDB.Eval(context.Background(), vodAcquireScript,
		[]string{vodSessPrefix + strconv.FormatInt(uid, 10)},
		now, config.VodHeartbeatTTL, config.VodMaxSessions, strconv.FormatInt(sessionID, 10)).Int()
	if err != nil {
		return nil, err
	}
	if acquired != 1 {
		dao.EndSession(sessionID, model.SessionEnded)
		return nil, ErrVodConcurrent
	}

	// ④ 发 ticket：TTL = 影片时长 + 缓冲
	ttl := m.Duration + config.VodTicketGrace
	if ttl < 3600 {
		ttl = 3600
	}
	exp := nowTime.Add(time.Duration(ttl) * time.Second).Unix()
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	info := vodTicketInfo{
		UID: uid, MovieID: movieID, SessionID: sessionID,
		RenditionID: rend.ID, DeviceID: deviceID, WmCode: wmCode, Exp: exp,
	}
	blob, _ := json.Marshal(info)
	if err := dao.RDB.Set(context.Background(), vodTicketPrefix+token, blob,
		time.Duration(ttl)*time.Second).Err(); err != nil {
		return nil, err
	}

	// HLS 线：为会话生成 per-session 密钥（存 Redis，随会话失效）
	if rend.DRMScheme == model.DRMSchemeAES128 {
		if _, err := ensureHLSKey(sessionID, ttl); err != nil {
			return nil, err
		}
	}

	res := &VodPlayResult{
		Ticket:      token,
		ManifestURL: "/vod/manifest?ticket=" + token,
		DRMScheme:   rend.DRMScheme,
	}
	if rend.DRMScheme == model.DRMSchemeClearKey {
		res.LicenseURL = "/vod/license"
	} else {
		res.LicenseURL = "/vod/key"
	}
	return res, nil
}

// ensureHLSKey 取/建 HLS per-session 密钥（返回 16 字节明文）
func ensureHLSKey(sessionID int64, ttl int) ([]byte, error) {
	key := vodHLSKeyPrefix + strconv.FormatInt(sessionID, 10)
	if hexKey, err := dao.RDB.Get(context.Background(), key).Result(); err == nil {
		if b, err := hex.DecodeString(hexKey); err == nil {
			return b, nil
		}
	}
	k, err := GenSessionKey()
	if err != nil {
		return nil, err
	}
	if err := dao.RDB.Set(context.Background(), key, hex.EncodeToString(k),
		time.Duration(ttl)*time.Second).Err(); err != nil {
		return nil, err
	}
	return k, nil
}

// LoadTicket 校验 ticket（含设备绑定）并返回携带的会话状态
func LoadTicket(token, deviceID string) (*vodTicketInfo, error) {
	if token == "" {
		return nil, ErrVodTicketInvalid
	}
	blob, err := dao.RDB.Get(context.Background(), vodTicketPrefix+token).Result()
	if err != nil {
		return nil, ErrVodTicketInvalid // 不存在 / 已过期 / 已吊销
	}
	var info vodTicketInfo
	if err := json.Unmarshal([]byte(blob), &info); err != nil {
		return nil, ErrVodTicketInvalid
	}
	if info.Exp < time.Now().Unix() {
		return nil, ErrVodTicketInvalid
	}
	// 绑定校验：ticket 关联 deviceId，不匹配 → 40302
	if deviceID != "" && info.DeviceID != "" && info.DeviceID != deviceID {
		return nil, ErrVodDeviceMismatch
	}
	return &info, nil
}

// VodHeartbeat 心跳续期：ZSET 刷新 score + 续 ticket TTL + 刷新会话心跳
func VodHeartbeat(token, deviceID string) (int64, error) {
	info, err := LoadTicket(token, deviceID)
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	ok, err := dao.RDB.Eval(context.Background(), vodHeartbeatScript,
		[]string{vodSessPrefix + strconv.FormatInt(info.UID, 10)},
		now, config.VodHeartbeatTTL, strconv.FormatInt(info.SessionID, 10)).Int()
	if err != nil {
		return 0, err
	}
	if ok != 1 {
		return 0, ErrVodTicketInvalid // 会话已被清理 / 吊销
	}
	// 续 ticket TTL 与会话心跳
	ttl := info.Exp - now
	if ttl < 60 {
		ttl = 60
	}
	dao.RDB.Expire(context.Background(), vodTicketPrefix+token, time.Duration(ttl)*time.Second)
	dao.TouchSessionHeartbeat(info.SessionID, time.Now())
	// HLS 密钥同步续期
	dao.RDB.Expire(context.Background(), vodHLSKeyPrefix+strconv.FormatInt(info.SessionID, 10),
		time.Duration(ttl)*time.Second)

	return info.Exp, nil
}

// VodStop 结束会话：ZREM + DEL ticket + 落库结束
func VodStop(token string) error {
	info, err := LoadTicket(token, "")
	if err != nil {
		return err
	}
	dao.RDB.ZRem(context.Background(), vodSessPrefix+strconv.FormatInt(info.UID, 10),
		strconv.FormatInt(info.SessionID, 10))
	dao.RDB.Del(context.Background(), vodTicketPrefix+token)
	dao.RDB.Del(context.Background(), vodHLSKeyPrefix+strconv.FormatInt(info.SessionID, 10))
	return dao.EndSession(info.SessionID, model.SessionEnded)
}

// VodRevoke 吊销某用户全部会话（发现盗号 / 越权时立刻踢人）
// 直接清 ZSET，让所有在途 ticket 的下一次校验失败
func VodRevoke(uid int64) error {
	dao.RDB.Del(context.Background(), vodSessPrefix+strconv.FormatInt(uid, 10))
	return dao.RevokeSessionsByUID(uid)
}

// WatermarkVariant 取某会话在某段应拿的变体位：返回 'a' 或 'b'
func WatermarkVariant(wmCode uint64, segNo int) byte {
	if segNo < 0 || segNo >= 64 {
		return 'a'
	}
	if (wmCode>>uint(segNo))&1 == 1 {
		return 'b'
	}
	return 'a'
}

// HLSKeyForSession 取会话的 HLS 密钥（分发时现加密用）
func HLSKeyForSession(sessionID int64) ([]byte, error) {
	hexKey, err := dao.RDB.Get(context.Background(), vodHLSKeyPrefix+strconv.FormatInt(sessionID, 10)).Result()
	if err != nil {
		return nil, fmt.Errorf("会话密钥不存在或已过期")
	}
	return hex.DecodeString(hexKey)
}

// SessionWatermark 取会话被分配的水印码（变体白名单校验用）
func SessionWatermark(sessionID int64) (uint64, error) {
	s, err := dao.RDB.Get(context.Background(), vodSessWMPrefix+strconv.FormatInt(sessionID, 10)).Result()
	if err != nil {
		return 0, ErrVodTicketInvalid // 会话不存在 / 已过期
	}
	return strconv.ParseUint(s, 10, 64)
}
