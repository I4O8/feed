package model

import "time"

// Feed 关注流扩散记录，对应 feed 表（退化版专用：feed 存 MySQL）
// 复合索引 (user_id, id)：支撑「刷某人的 feed，按 id 倒序」的游标分页
type Feed struct {
	ID        int64     `gorm:"primaryKey;index:idx_uid_id,priority:2"` // 自增主键，作游标/排序
	UserID    int64     `gorm:"not null;index:idx_uid_id,priority:1"`   // 该 feed 属于谁
	VideoID   int64     `gorm:"not null"`                               // 视频
	CreatedAt time.Time `gorm:"autoCreateTime"`                         // GORM 自动维护
}
