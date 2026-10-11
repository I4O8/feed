package handler

import (
	"fmt"
	"io"
	"strconv"

	"feed/model"
	"feed/service"

	"github.com/gin-gonic/gin"
)

// 点播模块：管理端 handler（Header X-Admin-Key 由 AdminMiddleware 校验）
// 文档：docs/点播模块.md 8.1

// AdminImportMovie POST /admin/movie/import
// 按 NFO 导入元数据（multipart 上传 .nfo 文件，或 JSON 传 nfoPath）
func AdminImportMovie(ctx *gin.Context) {
	ctype := ctx.GetHeader("Content-Type")

	// 方式一：multipart 上传 NFO 文件
	if fh, err := ctx.FormFile("nfo"); err == nil {
		f, err := fh.Open()
		if err != nil {
			Fail(ctx, CodeInternalError, "NFO 打开失败")
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			Fail(ctx, CodeInternalError, "NFO 读取失败")
			return
		}
		id, err := service.VodImportNfo(data, "")
		if err != nil {
			Fail(ctx, CodeParamError, "NFO 解析失败")
			fmt.Printf("nfo import err: %v\n", err)
			return
		}
		OK(ctx, gin.H{"movieId": id, "status": model.MovieStatusPacking})
		return
	}

	// 方式二：JSON 给服务器上已有 NFO 的路径
	_ = ctype
	var req struct {
		NfoPath string `json:"nfoPath"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.NfoPath == "" {
		Fail(ctx, CodeParamError, "需要上传 nfo 文件或提供 nfoPath")
		return
	}
	id, err := service.VodImportNfo(nil, req.NfoPath)
	if err != nil {
		Fail(ctx, CodeParamError, "NFO 解析失败")
		return
	}
	OK(ctx, gin.H{"movieId": id, "status": model.MovieStatusPacking})
}

// AdminCreateMovie POST /admin/movie
func AdminCreateMovie(ctx *gin.Context) {
	var m model.Movie
	if err := ctx.ShouldBindJSON(&m); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	id, err := service.VodCreateMovie(&m)
	if err != nil {
		Fail(ctx, CodeParamError, err.Error())
		return
	}
	OK(ctx, gin.H{"movieId": id})
}

// AdminUpdateMovie PUT /admin/movie/:id
func AdminUpdateMovie(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	var fields map[string]any
	if err := ctx.ShouldBindJSON(&fields); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodUpdateMovie(id, fields); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, nil)
}

// AdminDeleteMovie DELETE /admin/movie/:id
func AdminDeleteMovie(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodDeleteMovie(id); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, nil)
}

// AdminUploadSource POST /admin/movie/:id/source
// multipart 上传母版 A + B（两份成对，缺一不可）→ 触发异步打包
func AdminUploadSource(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	fileA, _ := ctx.FormFile("sourceA")
	fileB, _ := ctx.FormFile("sourceB")
	if fileA == nil || fileB == nil {
		Fail(ctx, CodeParamError, "母版 A 与 B 必须成对上传")
		return
	}
	if err := service.VodUploadSource(id, fileA, fileB); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, gin.H{"status": model.MovieStatusPacking})
}

// AdminMovieStatus GET /admin/movie/:id/status
func AdminMovieStatus(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	status, reason, segCount, err := service.VodMovieStatus(id)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, gin.H{"status": status, "failReason": reason, "segmentCount": segCount})
}
