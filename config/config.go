package config

import (
	"os"
)

var (
	Dsn       = getEnv("DSN", "gorm:gorm12345@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local")
	JWTSecret = getEnv("JWT_SECRET", "fONAdtaV+MOx4kBtsPgdtkJRiHqpu6Z0H+Jog4JOjn4=")
	UploadDir = getEnv("UPLOAD_DIR", "./data/avatar")
	VideoDir  = getEnv("VIDEO_DIR", "./data/video")
	CoverDir  = getEnv("COVER_DIR", VideoDir+"/cover")
	Port      = getEnv("PORT", "2222")
)

// 获取环境变量
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
