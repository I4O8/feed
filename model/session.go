package model

import "time"

// 播放会话状态
const (
	SessionActive    = 1 // 活跃
	SessionEnded     = 0 // 已结束
	SessionRevoked   = 2 // 被吊销（盗号 / 越权，立刻踢人）
)

// PlaybackSession 播放会话审计表：谁、什么片、什么设备、什么 IP、起止时间
// 文档：docs/点播模块.md 7.4 / 5.3 / 5.4
// 一期即包含 ip / device_id / user_agent / heartbeat_at —— 「为二期留口子」的典型做法
type PlaybackSession struct {
	ID            int64     `json:"id" gorm:"primaryKey"`                        // 主键自增
	UID           int64     `json:"uid" gorm:"not null;index:idx_sess_uid"`      // 观看用户
	MovieID       int64     `json:"movieId" gorm:"not null;index:idx_sess_movie"` // 影片
	RenditionID   int64     `json:"renditionId"`                                 // 选中的产物
	DeviceID      string    `json:"deviceId" gorm:"type:varchar(128)"`           // 浏览器指纹（复用 video 模块思路）
	IP            string    `json:"ip" gorm:"type:varchar(64)"`                  // 客户端 IP（二期 IP 跳变检测用）
	UserAgent     string    `json:"userAgent" gorm:"type:varchar(255)"`          // 二期风控用
	WatermarkCode uint64    `json:"watermarkCode" gorm:"index:idx_sess_wm"`      // 本次会话的 A/B 二进制码（位 i = 第 i 段取 A/B）
	Status        int8      `json:"status" gorm:"default:1"`                     // 1=活跃 0=已结束 2=被吊销
	StartedAt     time.Time `json:"startedAt" gorm:"autoCreateTime"`             // 会话开始
	HeartbeatAt   time.Time `json:"heartbeatAt"`                                 // 最后一次心跳
	EndedAt       *time.Time `json:"endedAt"`                                    // 会话结束
}
