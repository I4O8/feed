package handler

import (
	"fmt"
	"strings"

	"feed/service"

	"github.com/gin-gonic/gin"
)

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
