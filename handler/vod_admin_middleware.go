package handler

import (
	"feed/service"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware 点播模块管理端鉴权：请求头 X-Admin-Key，常量时间比较
// 文档：docs/点播模块.md 6.4 —— 独立中间件，不动现有账号模块（不引入 Role 字段）
func AdminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.GetHeader("X-Admin-Key")
		if !service.CheckAdminKey(key) {
			Fail(ctx, CodeAdminKeyInvalid, "admin 密钥无效")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
