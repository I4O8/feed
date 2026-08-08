package handler

import (
	"errors"
	"fmt"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// getUserIdFromCtx 从 JWT 中间件写入的上下文取当前用户 ID（复用 follow.go 的取法）
func getUserIdFromCtx(ctx *gin.Context) (int64, bool) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		return 0, false
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		return 0, false
	}
	return userID, true
}

// Like   POST /like/:id   需登录
func Like(ctx *gin.Context) {
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
	if err := service.Like(userID, videoID); err != nil {
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

// UnLike  DELETE /like/:id  需登录
func UnLike(ctx *gin.Context) {
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
	if err := service.UnLike(userID, videoID); err != nil {
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

// LikeStatus  GET /like/status?videoId=X  需登录
func LikeStatus(ctx *gin.Context) {
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
	liked, err := service.LikeStatus(userID, videoID)
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
	OK(ctx, gin.H{"liked": liked})
}
