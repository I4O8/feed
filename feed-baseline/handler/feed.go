package handler

import (
	"fmt"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// Feed GET /feed?offset=0&count=20  需登录，从 JWT 拿 userId
func Feed(ctx *gin.Context) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}

	var q struct {
		Offset int64 `form:"offset"`
		Count  int   `form:"count"`
	}
	q.Offset, q.Count = 0, 20 // 默认值
	if err := ctx.ShouldBindQuery(&q); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if q.Count > 50 {
		q.Count = 50 // 契约上限
	}

	videos, nextOffset, err := service.Feed(userID, q.Offset, q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("error:%v\n", err)
		return
	}
	OK(ctx, gin.H{
		"videos":     videos,
		"nextOffset": nextOffset,
	})
}
