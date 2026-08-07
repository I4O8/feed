package model

import "time"

// Favorites 收藏表：持久，可查"我的收藏"列表
// 文档：docs/点赞与收藏模块.md 第四节
type Favorites struct {
	ID        int64     `json:"id" gorm:"primaryKey"`                                                     // 主键自增
	UserID    int64     `json:"userId" gorm:"uniqueIndex:idx_fav;index:idx_fav_user;not null"`            // 收藏者
	VideoID   int64     `json:"videoId" gorm:"uniqueIndex:idx_fav;not null"`                              // 被藏视频
	CreatedAt time.Time // GORM 自动维护
}
