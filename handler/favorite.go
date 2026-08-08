package handler

import (
	"errors"
	"fmt"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// Fav   POST /fav/:id   需登录
func Fav(ctx *gin.Context) {
	userID, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	videoID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.Fav(userID, videoID); err != nil {
		switch {
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeUserNotFound, "视频不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}

// UnFav  DELETE /fav/:id  需登录
func UnFav(ctx *gin.Context) {
	userID, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	videoID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.UnFav(userID, videoID); err != nil {
		switch {
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeUserNotFound, "视频不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}

// FavStatus  GET /fav/status?videoId=X  需登录
func FavStatus(ctx *gin.Context) {
	userID, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	videoID, err := strconv.ParseInt(ctx.Query("videoId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	favored, err := service.FavStatus(userID, videoID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeUserNotFound, "视频不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, gin.H{"favored": favored})
}

// FavList  GET /fav/list?offset&count  需登录，从 JWT 拿 userId
func FavList(ctx *gin.Context) {
	userID, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
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

	videos, nextCursor, err := service.FavList(userID, q.Offset, q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("error:%v\n", err)
		return
	}
	OK(ctx, gin.H{
		"videos":     videos,
		"nextOffset": nextCursor,
	})
}
