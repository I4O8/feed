package model

import "time"

// MovieFavorite 电影收藏表（新建独立表，不动现有 favorites——video 用）
// 文档：docs/点播模块.md 7.5
type MovieFavorite struct {
	ID        int64     `json:"id" gorm:"primaryKey"`                                       // 主键自增
	UserID    int64     `json:"userId" gorm:"uniqueIndex:idx_mfav;not null"`                // 收藏者
	MovieID   int64     `json:"movieId" gorm:"uniqueIndex:idx_mfav;index:idx_mfav_user;not null"` // 被藏电影
	CreatedAt time.Time `json:"createdAt"`                                                  // GORM 自动维护
}
