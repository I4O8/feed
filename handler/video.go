package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

type uploadStruct struct {
	Video    *multipart.FileHeader `form:"file" binding:"required"`
	Cover    *multipart.FileHeader `form:"cover"`
	Title    string                `form:"title" binding:"required"`
	Describe string                `form:"describe"`
}

type list struct {
	Cursor int64 `form:"cursor"`
	Limit  int   `form:"limit"`
	UserId int64 `form:"userid"`
}

// isValidVideoType 视频格式校验，仅支持 mp4，mkv
func isValidVideoType(name string) bool {
	return filepath.Ext(name) == ".mp4" || filepath.Ext(name) == ".mkv"
}

// isVideoTooLarge 视频大小校验，上限 100MB
func isVideoTooLarge(size int64) bool {
	return size > 100*1024*1024
}

// isValidCoverType 封面格式校验，仅支持 jpg/png/webp
func isValidCoverType(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".jpg" || ext == ".png" || ext == ".webp"
}

// isCoverTooLarge 封面大小校验，上限 5MB
func isCoverTooLarge(size int64) bool {
	return size > 5*1024*1024
}

// UploadVideo 视频上传接口：解析 multipart → 校验 → 调 service → 返回
func UploadVideo(ctx *gin.Context) {
	// 鉴权：从中间件取当前登录用户
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

	// 绑定 multipart 参数
	var tmp uploadStruct
	if err := ctx.ShouldBind(&tmp); err != nil {
		Fail(ctx, CodeParamError, "参数错误")
		return
	}

	// 视频校验：格式 + 大小
	if !isValidVideoType(tmp.Video.Filename) {
		Fail(ctx, CodeFileTypeUnsupported, "不支持的视频类型")
		return
	}
	if isVideoTooLarge(tmp.Video.Size) {
		Fail(ctx, CodeFileTooLarge, "视频文件过大")
		return
	}

	// 封面（可选）：校验并读入内存；未传则 coverFile 保持 nil，由 service 抽帧
	var coverFile []byte
	var coverName string
	if tmp.Cover != nil {
		coverName = tmp.Cover.Filename
		if !isValidCoverType(coverName) {
			Fail(ctx, CodeFileTypeUnsupported, "不受支持的图片类型")
			return
		}
		if isCoverTooLarge(tmp.Cover.Size) {
			Fail(ctx, CodeFileTooLarge, "封面图片过大")
			return
		}
		tmpCover, err := tmp.Cover.Open()
		if err != nil {
			Fail(ctx, CodeInternalError, "文件打开失败")
			return
		}
		defer tmpCover.Close()
		coverFile, err = io.ReadAll(tmpCover)
		if err != nil {
			Fail(ctx, CodeInternalError, "文件读取失败")
			return
		}
	}

	// 视频流式传给 service
	videoFile, err := tmp.Video.Open()
	if err != nil {
		Fail(ctx, CodeInternalError, "视频打开失败")
		return
	}
	defer videoFile.Close()

	// 调 service，映射业务错误
	videoID, videoURL, coverURL, duration, err := service.UploadVideo(videoFile, tmp.Video.Filename, coverFile, coverName, userID, tmp.Title, tmp.Describe)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCoverCaptureFailed):
			Fail(ctx, CodeInternalError, "封面获取失败")
		case errors.Is(err, service.ErrDamagedVideo):
			Fail(ctx, CodeParamError, "损坏/不完整的视频")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
		}
		fmt.Printf("upload video err: %v\n", err)
		return
	}
	OK(ctx, gin.H{
		"videoId":  videoID,
		"videoUrl": videoURL,
		"coverUrl": coverURL,
		"duration": duration,
	})
}

// GetDetail 视频详情接口：按 videoId 返回单条视频（含发布者信息）
func GetDetail(ctx *gin.Context) {
	videoID, err := strconv.ParseInt(ctx.Query("videoId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	video, err := service.GetDetail(videoID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeResourceNotFound, "视频资源不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, video)
}

// ListVideo 视频列表接口：游标分页，可选按发布者过滤（userId 不传 = 全站）
func ListVideo(ctx *gin.Context) {
	// 绑定 query 参数（cursor 默认 0，limit 默认 20）
	tmp := list{Cursor: 0, Limit: 20}
	if err := ctx.ShouldBindQuery(&tmp); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	// limit 上限 50（契约）
	if tmp.Limit > 50 {
		tmp.Limit = 50
	}

	videos, nextCursor, err := service.List(tmp.UserId, tmp.Cursor, tmp.Limit)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("error:%v\n", err)
		return
	}
	OK(ctx, gin.H{
		"videos":     videos,
		"nextCursor": nextCursor,
	})
}

// DeleteVideo 删除视频
func DeleteVideo(ctx *gin.Context) {
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
	videoID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.DeleteVideo(userID, videoID); err != nil {
		switch {
		case errors.Is(err, service.ErrPermissionDenied):
			Fail(ctx, CodeForbidden, "没有足够的权限")
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeResourceNotFound, "内容已被删除")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}

// ReportPlay 播放上报接口：公开（无需登录），播放量 +1（本模块不做去重）
func ReportPlay(ctx *gin.Context) {
	var req struct {
		VideoID int64 `json:"videoId" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.ReportPlay(req.VideoID); err != nil {
		switch {
		case errors.Is(err, service.ErrVideoNotFound):
			Fail(ctx, CodeResourceNotFound, "视频不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}
