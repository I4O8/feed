package model

import "time"

// 封装形态与加密方案
const (
	ContainerHLS  = "hls"  // HLS（m3u8 + 分片）
	ContainerDASH = "dash" // DASH（MPD + 分片）

	DRMSchemeAES128    = "AES128"    // HLS 辅线：AES-128，per-session key，分发时现加密
	DRMSchemeClearKey  = "CLEARKEY"  // DASH 主线：CENC 加密 + ClearKey license
	DRMSchemeNone      = "NONE"      // 不加密（调试用）
	DRMSchemeAES128Tag = "AES-128"   // m3u8 中 #EXT-X-KEY 的 METHOD 取值
)

// MovieRendition 产物清单表：一行 = 一个可播放产物（码率 × 加密方案）
// 文档：docs/点播模块.md 7.2
// 播放时由闸门按客户端能力与偏好选择合适的一条，返回对应 manifest
type MovieRendition struct {
	ID               int64     `json:"id" gorm:"primaryKey"`                                    // 主键自增
	MovieID          int64     `json:"movieId" gorm:"not null;index:idx_rendition_pick"`        // 所属电影
	Resolution       string    `json:"resolution" gorm:"not null;type:varchar(32)"`             // 如 1080p
	Bitrate          int       `json:"bitrate"`                                                 // kbps
	Codec            string    `json:"codec" gorm:"type:varchar(32)"`                           // 如 h264
	Container        string    `json:"container" gorm:"not null;type:varchar(16);index:idx_rendition_pick"` // hls / dash
	DRMScheme        string    `json:"drmScheme" gorm:"not null;type:varchar(16);index:idx_rendition_pick"` // AES128 / CLEARKEY / NONE
	PlaylistPath     string    `json:"playlistPath" gorm:"not null;type:varchar(255)"`          // manifest 模板路径（逐用户生成时填码）
	SegmentPatternA  string    `json:"-" gorm:"type:varchar(255)"`                              // A 变体分片路径模式（含 %d 段号占位）
	SegmentPatternB  string    `json:"-" gorm:"type:varchar(255)"`                              // B 变体分片路径模式
	ContentKeyID     int64     `json:"contentKeyId"`                                            // 关联 content_key；A/B 共用同一把 key
	CreatedAt        time.Time `json:"createdAt" gorm:"autoCreateTime"`                         // GORM 自动维护
}
