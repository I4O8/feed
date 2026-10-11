package handler

import (
	"fmt"
	"strings"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 鉴权中间件：校验 Bearer token，把 userId 写入上下文
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			Fail(ctx, CodeTokenMissingOrExpired, "损坏的Token")
			ctx.Abort()
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		userID, err := service.Parse(token)
		if err != nil {
			Fail(ctx, CodeTokenMissingOrExpired, "Token无效或已过期")
			fmt.Printf("error:%v\n", err)
			ctx.Abort()
			return
		}

		ctx.Set("UserId", userID)
		ctx.Next()
	}
}

// OptionalAuth 可选登录中间件（点播详情用）：带了有效 token 就写入 UserId，
// 没带或无效也放行 —— 详情页对未登录用户同样可看，只是不返回「是否收藏 / 进度」
func OptionalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			if userID, err := service.Parse(strings.TrimPrefix(token, "Bearer ")); err == nil {
				ctx.Set("UserId", userID)
			}
		}
		ctx.Next()
	}
}
