package config

import (
	"os"
	"strconv"
)

var (
	Dsn       = getEnv("DSN", "gorm:gorm12345@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local")
	JWTSecret = getEnv("JWT_SECRET", "fONAdtaV+MOx4kBtsPgdtkJRiHqpu6Z0H+Jog4JOjn4=")
	UploadDir = getEnv("UPLOAD_DIR", "./data/avatar")
	VideoDir  = getEnv("VIDEO_DIR", "./data/video")
	CoverDir  = getEnv("COVER_DIR", VideoDir+"/cover")
	Port      = getEnv("PORT", "2222")
	RedisAddr = getEnv("REDIS_ADDR", "localhost:6379")

	// ── 点播（VOD）模块 ── 文档：docs/点播模块.md 六、访问控制 / 7.7 海报本地化
	VodDir       = getEnv("VOD_DIR", "./data/vod")            // 分片 / manifest / 母版根目录
	PosterDir    = getEnv("POSTER_DIR", "./data/poster")      // 本地化海报目录（复用 /uploads 静态方案）
	NfoDir       = getEnv("NFO_DIR", "./data/nfo")            // NFO 刮削信息落盘目录
	VodURLSecret = getEnv("VOD_URL_SECRET", "dev-vod-url-secret-change-me")   // 签名 URL 的 HMAC 密钥
	VodKEK       = getEnv("VOD_KEK", "dev-vod-kek-32bytes-change-me!!")       // 内容密钥信封加密主密钥
	VodAdminKey  = getEnv("VOD_ADMIN_KEY", "dev-admin-key")                   // admin 密钥（X-Admin-Key）
	VodMaxSessions   = getEnvInt("VOD_MAX_SESSIONS", 2)                       // 单用户最大并发播放会话
	VodSegmentSigTTL = getEnvInt("VOD_SEGMENT_SIG_TTL", 3600)                 // 分片签名有效期（秒）
	VodTicketGrace   = getEnvInt("VOD_TICKET_GRACE", 1800)                    // ticket TTL = 影片时长 + 缓冲（秒）
	VodHeartbeatTTL  = getEnvInt("VOD_HEARTBEAT_TTL", 90)                     // 心跳容忍窗口（秒），超时判会话已死
	VodTargetSegments = getEnvInt("VOD_TARGET_SEGMENTS", 32)                  // 水印段数目标（码长 ≤ 64）

	// ── 打包工具链（见 docs 实现记录：HLS 线纯 ffmpeg；DASH CENC 线用 GPAC MP4Box）──
	FfmpegBin  = getEnv("FFMPEG_BIN", "ffmpeg")   // ffmpeg：remux + HLS 分片
	FfprobeBin = getEnv("FFPROBE_BIN", "ffprobe") // ffprobe：逐片 IDR 分析
	Mp4BoxBin  = getEnv("MP4BOX_BIN", "MP4Box")   // MP4Box：CENC 加密 + DASH 分片
	// PublicBaseURL 对外基地址：MPD 里的 license Laurl 要用绝对 URL，
	// 否则老版 shaka 的 EME license 请求可能因相对地址解析失败。部署到域名时改这里。
	PublicBaseURL = getEnv("PUBLIC_BASE_URL", "http://localhost:2222")
)

// 获取环境变量
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt 获取整型环境变量，解析失败回退默认值
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
