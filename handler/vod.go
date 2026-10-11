package handler

import (
	"errors"
	"fmt"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// 点播模块：播放链路 handler
// 文档：docs/点播模块.md 8.3 播放（登录）

// vodFail 点播错误 → 错误码 的统一映射
func vodFail(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVodMovieNotFound):
		Fail(ctx, CodeVodMovieNotFound, "电影不存在")
	case errors.Is(err, service.ErrVodNotReady):
		Fail(ctx, CodeVodNotReady, "影片准备中")
	case errors.Is(err, service.ErrVodConcurrent):
		Fail(ctx, CodeVodConcurrent, "同时观看设备数已达上限")
	case errors.Is(err, service.ErrVodTicketInvalid):
		Fail(ctx, CodeVodTicketInvalid, "播放凭证无效或已过期")
	case errors.Is(err, service.ErrVodDeviceMismatch):
		Fail(ctx, CodeVodDeviceMismatch, "设备不匹配，请重新开播")
	case errors.Is(err, service.ErrVodVariantForbidden):
		Fail(ctx, CodeVodVariantForbidden, "变体请求越权")
	case errors.Is(err, service.ErrVodSigInvalid):
		Fail(ctx, CodeVodSigInvalid, "签名无效或已过期")
	default:
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("vod error: %v\n", err)
	}
}

// VodPlay POST /vod/play 需登录
// 开播：鉴权(中间件) + 并发检查 + 发 ticket + manifest URL + 分配水印码
func VodPlay(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	var req struct {
		MovieID    int64  `json:"movieId" binding:"required"`
		DeviceID   string `json:"deviceId" binding:"required"`
		DRMScheme  string `json:"drmScheme"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	res, err := service.VodPlay(uid, req.MovieID, req.DeviceID, req.DRMScheme,
		ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, res)
}

// VodManifest GET /vod/manifest?ticket=...
// 返回 per-user 动态 manifest（HLS 为 m3u8 文本 / DASH 为 MPD 文本），不是 JSON
func VodManifest(ctx *gin.Context) {
	ticket := ctx.Query("ticket")
	body, ctype, err := service.VodManifest(ticket)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	ctx.Data(200, ctype, []byte(body))
}

// VodKey GET /vod/key?ticket=&deviceId=  ★闸门★
// HLS AES-128 密钥：返回 16 字节原始密钥（octet-stream）
func VodKey(ctx *gin.Context) {
	ticket := ctx.Query("ticket")
	deviceID := ctx.Query("deviceId")
	key, err := service.VodKey(ticket, deviceID)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	ctx.Data(200, "application/octet-stream", key)
}

// VodLicense POST /vod/license?ticket=&deviceId=  ★闸门★
// DASH ClearKey license：返回 W3C ClearKey JSON
func VodLicense(ctx *gin.Context) {
	ticket := ctx.Query("ticket")
	deviceID := ctx.Query("deviceId")
	resp, err := service.VodLicense(ticket, deviceID)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	ctx.Data(200, "application/json", resp)
}

// VodSegment GET /vod/segment?ticket=&m=&r=&s=&n=&v=&exp=&sig=&kind=
// 分片分发：验签 + 变体白名单 → HLS 线现加密 / CENC 线直出密文
func VodSegment(ctx *gin.Context) {
	q := ctx.Request.URL.Query()
	atoi := func(k string) int64 { n, _ := strconv.ParseInt(q.Get(k), 10, 64); return n }

	ticket := q.Get("ticket")
	kind := q.Get("kind")
	movieID := atoi("m")
	renditionID := atoi("r")
	sessionID := atoi("s")
	segNo := int(atoi("n"))
	exp := atoi("exp")
	sig := q.Get("sig")

	var variant byte = 'a'
	if v := q.Get("v"); len(v) > 0 {
		variant = v[0]
	}

	data, err := service.VodSegment(ticket, movieID, renditionID, sessionID, segNo, variant, kind, exp, sig)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	ctx.Data(200, "video/mp2t", data) // 统一按 TS 返回（CENC 分片客户端按 Content-Type 不敏感）
}

// VodHeartbeat POST /vod/heartbeat
func VodHeartbeat(ctx *gin.Context) {
	var req struct {
		Ticket   string `json:"ticket" binding:"required"`
		DeviceID string `json:"deviceId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	exp, err := service.VodHeartbeat(req.Ticket, req.DeviceID)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, gin.H{"ok": true, "expiresAt": exp})
}

// VodStop POST /vod/stop
func VodStop(ctx *gin.Context) {
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodStop(req.Ticket); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, gin.H{"ok": true})
}
