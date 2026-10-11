package main

import (
	"feed/config"
	"feed/dao"
	"feed/handler"
	"feed/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	dao.InitSQL()
	dao.InitRedis()
	service.StartCountFlush()
	service.StartLikeFlush()
	if err := service.EnsureVodDirs(); err != nil {
		panic("点播目录初始化失败: " + err.Error())
	}
	service.StartMovieCountFlush()
	if err := service.InitFeedMQ(); err != nil {
		panic("RabbitMQ 连接失败: " + err.Error())
	}
	service.StartFanoutConsumer()
	if err := service.InitVodMQ(); err != nil {
		panic("点播打包队列初始化失败: " + err.Error())
	}
	service.StartPackConsumer()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"}, // 前端 dev 地址
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Admin-Key"},
	}))
	r.MaxMultipartMemory = 2 << 30 // 母版为高码率源片，放宽到 2GB（gin 默认仅 32MB）
	r.Static("/uploads", "./data")
	r.POST("/user/register", handler.Register)
	r.POST("/user/login", handler.Login)
	r.POST("/user/changePassword", handler.AuthMiddleware(), handler.ChangePassword)
	r.POST("/user/uploadAvatar", handler.AuthMiddleware(), handler.UploadAvatar)
	r.GET("/user/getProfile", handler.GetProfile)
	r.POST("/user/updateProfile", handler.AuthMiddleware(), handler.UpdateProfile)
	r.POST("/video/upload", handler.AuthMiddleware(), handler.UploadVideo)
	r.GET("/video/detail", handler.GetDetail)
	r.GET("/video/list", handler.ListVideo)
	r.DELETE("/video/:id", handler.AuthMiddleware(), handler.DeleteVideo)
	r.POST("/video/reportPlay", handler.ReportPlay)
	r.POST("/follow/:id", handler.AuthMiddleware(), handler.Follow)
	r.DELETE("/follow/:id", handler.AuthMiddleware(), handler.Unfollow)
	r.GET("/follow/list", handler.AuthMiddleware(), handler.FollowList)
	r.GET("/follow/fans", handler.AuthMiddleware(), handler.FansList)
	r.GET("/feed", handler.AuthMiddleware(), handler.Feed)
	r.POST("/like/:id", handler.AuthMiddleware(), handler.Like)
	r.DELETE("/like/:id", handler.AuthMiddleware(), handler.UnLike)
	r.GET("/like/status", handler.AuthMiddleware(), handler.LikeStatus)
	r.POST("/fav/:id", handler.AuthMiddleware(), handler.Fav)
	r.DELETE("/fav/:id", handler.AuthMiddleware(), handler.UnFav)
	r.GET("/fav/status", handler.AuthMiddleware(), handler.FavStatus)
	r.GET("/fav/list", handler.AuthMiddleware(), handler.FavList)

	// ───────────────── 点播（VOD）模块 ─────────────────
	// 文档：docs/点播模块.md 八、接口清单

	// 管理端（Header X-Admin-Key）
	admin := r.Group("/admin", handler.AdminMiddleware())
	{
		admin.POST("/movie/import", handler.AdminImportMovie)
		admin.POST("/movie", handler.AdminCreateMovie)
		admin.PUT("/movie/:id", handler.AdminUpdateMovie)
		admin.DELETE("/movie/:id", handler.AdminDeleteMovie)
		admin.POST("/movie/:id/source", handler.AdminUploadSource)
		admin.GET("/movie/:id/status", handler.AdminMovieStatus)
	}

	// 浏览（公开；detail 可选登录以返回收藏/进度）
	r.GET("/movie/list", handler.MovieList)
	r.GET("/movie/detail", handler.OptionalAuth(), handler.MovieDetail)
	r.GET("/movie/search", handler.MovieSearch)
	r.POST("/movie/reportPlay", handler.MovieReportPlay)

	// 播放链路（登录）
	r.POST("/vod/play", handler.AuthMiddleware(), handler.VodPlay)
	r.GET("/vod/manifest", handler.VodManifest)
	r.GET("/vod/key", handler.VodKey)
	r.POST("/vod/license", handler.VodLicense)
	r.GET("/vod/segment", handler.VodSegment)
	r.POST("/vod/heartbeat", handler.VodHeartbeat)
	r.POST("/vod/stop", handler.VodStop)

	// 观看功能（登录）
	r.POST("/movie/reportProgress", handler.AuthMiddleware(), handler.MovieReportProgress)
	r.GET("/movie/progress", handler.AuthMiddleware(), handler.MovieProgress)
	r.GET("/movie/history", handler.AuthMiddleware(), handler.MovieHistory)
	r.POST("/movie/fav/:id", handler.AuthMiddleware(), handler.MovieFav)
	r.DELETE("/movie/fav/:id", handler.AuthMiddleware(), handler.MovieUnFav)
	r.GET("/movie/fav/status", handler.AuthMiddleware(), handler.MovieFavStatus)
	r.GET("/movie/fav/list", handler.AuthMiddleware(), handler.MovieFavList)

	r.Run(":" + config.Port)
}
