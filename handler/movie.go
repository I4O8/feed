package handler

import (
	"fmt"
	"strconv"

	"feed/service"

	"github.com/gin-gonic/gin"
)

// 点播模块：片库浏览 / 观看功能 handler
// 文档：docs/点播模块.md 8.2 / 8.4

// movieListQuery 列表公共分页参数
type movieListQuery struct {
	Sort   string `form:"sort"`
	Genre  string `form:"genre"`
	Offset int    `form:"offset"`
	Count  int    `form:"count"`
}

// MovieList GET /movie/list?sort=&genre=&offset=&count=
func MovieList(ctx *gin.Context) {
	q := movieListQuery{Sort: "latest", Count: 20}
	if err := ctx.ShouldBindQuery(&q); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if q.Count > 50 {
		q.Count = 50
	}
	if q.Count < 1 {
		q.Count = 20
	}
	movies, next, err := service.VodList(q.Sort, q.Genre, q.Offset, q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("movie list err: %v\n", err)
		return
	}
	OK(ctx, gin.H{"movies": movies, "nextOffset": next})
}

// MovieDetail GET /movie/detail?movieId=
// 登录时额外返回「是否收藏 + 我的观看进度」
func MovieDetail(ctx *gin.Context) {
	movieID, err := strconv.ParseInt(ctx.Query("movieId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	var uid int64
	if u, ok := getUserIdFromCtx(ctx); ok {
		uid = u
	}
	m, err := service.VodDetail(movieID, uid)
	if err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, m)
}

// MovieSearch GET /movie/search?keyword=&offset=&count=
func MovieSearch(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
	if keyword == "" {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	q := movieListQuery{Count: 20}
	if err := ctx.ShouldBindQuery(&q); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if q.Count > 50 {
		q.Count = 50
	}
	movies, next, err := service.VodSearch(keyword, q.Offset, q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, gin.H{"movies": movies, "nextOffset": next})
}

// MovieReportPlay POST /movie/reportPlay — 公开（无需登录）
func MovieReportPlay(ctx *gin.Context) {
	var req struct {
		MovieID int64 `json:"movieId" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodReportPlay(req.MovieID); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, nil)
}

// MovieReportProgress POST /movie/reportProgress 需登录
// 前端节流上报（每 10~15s 一次 + 暂停/离开时补一次），不每次 timeupdate 都发
func MovieReportProgress(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	var req struct {
		MovieID  int64 `json:"movieId" binding:"required"`
		Position int   `json:"position"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodReportProgress(uid, req.MovieID, req.Position); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, nil)
}

// MovieProgress GET /movie/progress?movieId= 需登录
func MovieProgress(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	movieID, err := strconv.ParseInt(ctx.Query("movieId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	pos, err := service.VodProgress(uid, movieID)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, gin.H{"position": pos})
}

// MovieHistory GET /movie/history?cursor=&count= 需登录
func MovieHistory(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	q := movieListQuery{Count: 20}
	if err := ctx.ShouldBindQuery(&q); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if q.Count > 50 {
		q.Count = 50
	}
	cursor, _ := strconv.ParseInt(ctx.Query("cursor"), 10, 64)
	items, next, err := service.VodHistory(uid, cursor, q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, gin.H{"items": items, "nextCursor": next})
}

// MovieFav POST /movie/fav/:id 需登录
func MovieFav(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodFav(uid, id); err != nil {
		vodFail(ctx, err)
		return
	}
	OK(ctx, nil)
}

// MovieUnFav DELETE /movie/fav/:id 需登录
func MovieUnFav(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if err := service.VodUnFav(uid, id); err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, nil)
}

// MovieFavStatus GET /movie/fav/status?movieId= 需登录
func MovieFavStatus(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	movieID, err := strconv.ParseInt(ctx.Query("movieId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	favored, err := service.VodFavStatus(uid, movieID)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, gin.H{"favored": favored})
}

// MovieFavList GET /movie/fav/list?offset=&count= 需登录（offset 复用为 id 游标）
func MovieFavList(ctx *gin.Context) {
	uid, ok := getUserIdFromCtx(ctx)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	q := movieListQuery{Count: 20}
	if err := ctx.ShouldBindQuery(&q); err != nil {
		Fail(ctx, CodeParamError, "错误的请求参数")
		return
	}
	if q.Count > 50 {
		q.Count = 50
	}
	movies, next, err := service.VodFavList(uid, int64(q.Offset), q.Count)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, gin.H{"movies": movies, "nextCursor": next})
}
