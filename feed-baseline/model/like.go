package model

import "time"

// Likes 点赞表：持久去重记录 + 计数权威（不提供"已赞列表"查询，点赞关系短期以 Redis 为准）
// 文档：docs/点赞与收藏模块.md 第四节
type Likes struct {
	ID        int64     `json:"id" gorm:"primaryKey"`                                                     // 主键自增
	UserID    int64     `json:"userId" gorm:"uniqueIndex:idx_like;not null"`                              // 点赞者
	VideoID   int64     `json:"videoId" gorm:"uniqueIndex:idx_like;index:idx_like_video;not null"`        // 被赞视频
	CreatedAt time.Time // GORM 自动维护
}
