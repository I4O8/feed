package service

import (
	"encoding/base64"
	"fmt"
	"os"

	"feed/dao"
	"feed/model"
)

// 点播模块：密钥 / license 端点 + 分片分发
// 文档：docs/点播模块.md 6.2 密钥端点（闸门）/ 6.1 签名 URL / 6.5 变体访问控制
// 这两个端点（key / license）是本模块唯一的策略执行点 —— 并发、设备、吊销、时效全在这里收敛

// VodManifest 生成 per-user manifest
// 返回 (内容, 内容类型)；HLS 返回 m3u8 文本，DASH 返回 MPD 文本
func VodManifest(ticket string) (body string, contentType string, err error) {
	info, err := LoadTicket(ticket, "")
	if err != nil {
		return "", "", err
	}
	m, err := dao.FindMovieByID(info.MovieID)
	if err != nil || m == nil {
		return "", "", ErrVodMovieNotFound
	}
	rend, err := dao.FindRenditionByID(info.RenditionID)
	if err != nil || rend == nil {
		return "", "", ErrVodNotReady
	}

	starts := LoadSegmentStarts(m)
	totalDur := float64(m.Duration)
	segs := manifestSegs(ticket, info.MovieID, rend.ID, info.SessionID, info.WmCode, starts, totalDur)

	if rend.Container == model.ContainerHLS {
		return BuildHLSPlaylist(ticket, segs), "application/vnd.apple.mpegurl", nil
	}

	// DASH：取内容密钥的 kid 填进 ContentProtection
	kidHex := ""
	if ck, _ := dao.FindContentKeyByID(rend.ContentKeyID); ck != nil {
		kidHex = ck.KeyID
	}
	return BuildDASHMPD(ticket, segs, kidHex, totalDur), "application/dash+xml", nil
}

// VodKey HLS AES-128 密钥端点（闸门）
// 验 ticket + 会话存活 + deviceId → 返回本会话的 per-session 密钥（16 字节原始）
func VodKey(ticket, deviceID string) ([]byte, error) {
	info, err := LoadTicket(ticket, deviceID)
	if err != nil {
		return nil, err
	}
	rend, err := dao.FindRenditionByID(info.RenditionID)
	if err != nil || rend == nil {
		return nil, ErrVodNotReady
	}
	if rend.DRMScheme != model.DRMSchemeAES128 {
		return nil, fmt.Errorf("该产物不是 AES-128 加密线")
	}
	return HLSKeyForSession(info.SessionID)
}

// VodLicense DASH ClearKey license 端点（闸门）
// 验 ticket → 解信封拿内容密钥 → 返回 W3C ClearKey JSON
// 诚实边界（文档 1.1）：ClearKey 的 license 响应里密钥是明文 base64 —— 这是标准如此，
// 不承诺抗密钥提取；本端点的价值在「谁能拿到」的闸门，不在「拿到后能不能保护」
func VodLicense(ticket, deviceID string) ([]byte, error) {
	info, err := LoadTicket(ticket, deviceID)
	if err != nil {
		return nil, err
	}
	rend, err := dao.FindRenditionByID(info.RenditionID)
	if err != nil || rend == nil {
		return nil, ErrVodNotReady
	}
	ck, err := dao.FindContentKeyByID(rend.ContentKeyID)
	if err != nil || ck == nil {
		return nil, ErrVodNotReady
	}
	plain, err := OpenContentKey(ck.KeyValue, ck.Nonce)
	if err != nil {
		return nil, err
	}
	// W3C ClearKey JSON：{"keys":[{"kty":"oct","k":"<b64url>","kid":"<b64url>"}]}
	// kid 也要 base64url（与 manifest 里的 default_KID 是同一 KID 的两种编码）
	kidRaw := hexToBytes(ck.KeyID)
	resp := fmt.Sprintf(`{"keys":[{"kty":"oct","k":"%s","kid":"%s"}]}`,
		base64.RawURLEncoding.EncodeToString(plain),
		base64.RawURLEncoding.EncodeToString(kidRaw))
	return []byte(resp), nil
}

// VodSegment 分片分发
// kind=init → DASH 初始化段（带独立签名）
// 否则 → 媒体分片：验签 + ★变体白名单★ → HLS 线现加密 / CENC 线直出密文
func VodSegment(ticket string, movieID, renditionID, sessionID int64, segNo int,
	variant byte, kind string, exp int64, sig string) (data []byte, err error) {

	// ① 验 ticket：确认会话真实存在（防伪造 sessionId / 防裸取分片）
	info, err := LoadTicket(ticket, "")
	if err != nil {
		return nil, err
	}

	// ② init 段：只验签名 + 直出（init 与变体、与具体段都无关，URL 不带 m/r/s）
	if kind == "init" {
		if !VerifySegmentSig(-1, -1, -1, -1, 'a', exp, sig) {
			return nil, ErrVodSigInvalid
		}
		rend, err := dao.FindRenditionByID(info.RenditionID)
		if err != nil || rend == nil {
			return nil, ErrVodNotReady
		}
		p := initFile(rend)
		if p == "" {
			return nil, ErrVodNotReady
		}
		return os.ReadFile(p)
	}

	// ③ 媒体分片：ticket 与请求的会话/影片必须一致
	if info.SessionID != sessionID || info.MovieID != movieID {
		return nil, ErrVodSigInvalid
	}
	rend, err := dao.FindRenditionByID(renditionID)
	if err != nil || rend == nil {
		return nil, ErrVodNotReady
	}
	// 验签名（签名串覆盖变体位）
	if !VerifySegmentSig(movieID, renditionID, sessionID, segNo, variant, exp, sig) {
		return nil, ErrVodSigInvalid
	}

	// ④ ★变体白名单★（文档 6.5 第 3 条）：请求的变体位必须 == 本会话被分配的那个
	wmCode, err := SessionWatermark(sessionID)
	if err != nil {
		return nil, err
	}
	want := WatermarkVariant(wmCode, segNo)
	if variant != want {
		// 越权采集信号（6.5 第 4 条）：记异常，供告警 / 限速
		RecordVariantAbuse(sessionID)
		return nil, ErrVodVariantForbidden
	}

	// ⑤ 按产物路径模式定位文件
	path := segmentFile(rend, segNo, variant)
	plain, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrVodNotReady
	}

	// ⑥ HLS 线：明文分片 → 用本会话密钥现加密（per-session key）
	//    CENC 线：磁盘上就是密文，直接返回
	if rend.DRMScheme == model.DRMSchemeAES128 {
		key, err := HLSKeyForSession(sessionID)
		if err != nil {
			return nil, err
		}
		return EncryptHLSSegment(plain, key, segNo)
	}
	return plain, nil
}

// hexToBytes hex 串 → 字节（不引额外包，手写足够）
func hexToBytes(s string) []byte {
	out := make([]byte, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		out[i/2] = hexVal(s[i])<<4 | hexVal(s[i+1])
	}
	return out
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
