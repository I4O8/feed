package handler

import (
	"errors"
	"fmt"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// temp.go → 完成后重命名 follow.go
// 对应文档：docs/feed模块.md 第六节接口 1-4

// Follow   POST /follow/:id     需登录，不能关注自己(40003，handler 层拦)
func Follow(ctx *gin.Context) {
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
	targetID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if targetID == userID {
		Fail(ctx, CodeSelfFollow, "不允许关注自己")
		return
	}
	if err := service.Follow(userID, targetID); err != nil {
		switch {
		case errors.Is(err, service.ErrTargetUserNotExist):
			Fail(ctx, CodeUserNotFound, "目标用户不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}

// Unfollow DELETE /follow/:id   需登录，未关注幂等
func Unfollow(ctx *gin.Context) {
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
	targetID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if targetID == userID {
		Fail(ctx, CodeSelfFollow, "不允许取消关注自己")
		return
	}
	if err := service.Unfollow(userID, targetID); err != nil {
		switch {
		case errors.Is(err, service.ErrTargetUserNotExist):
			Fail(ctx, CodeUserNotFound, "目标用户不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)

		}
		return
	}
	OK(ctx, nil)
}

// FollowList GET /follow/list?userId=X  公开
func FollowList(ctx *gin.Context) {
	// 我不想未登录用户能查别人，所以加了权鉴
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	_, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}

	targetID, err := strconv.ParseInt(ctx.Query("userId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	list, err := service.FollowList(targetID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTargetUserNotExist):
			Fail(ctx, CodeUserNotFound, "目标用户不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, gin.H{
		"users": list,
	})
}

// FansList   GET /follow/fans?userId=X  公开
func FansList(ctx *gin.Context) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	_, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}

	targetID, err := strconv.ParseInt(ctx.Query("userId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}

	list, err := service.FansList(targetID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTargetUserNotExist):
			Fail(ctx, CodeUserNotFound, "目标用户不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, gin.H{
		"users": list,
	})
}

//   - 错误码 40003（不能关注自己）需要在 response.go 新增常量
//   - 需登录接口从 ctx.Get("UserId") 取当前用户（参考 handler/video.go）
//   - 错误映射：ErrTargetUserNotExist → CodeUserNotFound(40400)
//   - 列表响应：{"users": [...]}；path 参数用 ctx.Param("id") 解析
