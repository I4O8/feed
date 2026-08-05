package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess               = 0     // 成功
	CodeParamError            = 40000 // 参数错误 / 缺字段
	CodeLoginNameInvalid      = 40001 // 登录名格式不合法（非手机号也非邮箱）
	CodePasswordLenInvalid    = 40002 // 密码长度不合法（需 6-32 位）
	CodeTokenMissingOrExpired = 40100 // 未登录：token 缺失 / 无效 / 过期
	CodeLoginNameOrPwdWrong   = 40101 // 登录名或密码错误
	CodeOldPasswordWrong      = 40102 // 旧密码错误
	CodeAccountBanned         = 40103 // 账号被封禁
	CodeForbidden             = 40300 // 无权操作（如删除他人视频）
	CodeUserNotFound          = 40400 // 资源不存在（用户 / 视频）
	CodeResourceNotFound      = 40400 // 别名：资源不存在（语义化）
	CodeLoginNameRegistered   = 40900 // 登录名已被注册
	CodeFileTooLarge          = 41300 // 上传超过大小限制（头像 5MB / 视频 100MB / 封面 5MB）
	CodeFileTypeUnsupported   = 41500 // 文件类型不支持（图片仅 jpg/png/webp，视频仅 mp4）
	CodeTooManyRequests       = 42900 // 请求过于频繁（如注册 IP 限流）
	CodeInternalError         = 50000 // 服务器内部错误（兜底）
)

// OK 统一的 成功 回复函数
func OK(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// Fail 统一的 失败 回复函数
func Fail(ctx *gin.Context, code int, message string) {
	ctx.JSON(http.StatusOK, gin.H{"code": code, "message": message})
}
