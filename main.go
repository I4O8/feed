package main

import (
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
	if err := service.InitFeedMQ(); err != nil {
		panic("RabbitMQ 连接失败: " + err.Error())
	}
	service.StartFanoutConsumer()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"}, // 前端 dev 地址
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))
	r.MaxMultipartMemory = 128 << 20 // 支持最大 100MB 的视频上传（gin 默认仅 32MB）
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
	r.Run(":2222")
}
