package model

import "time"

// MovieProgress 观看进度表：断点续播 + 观看历史
// 文档：docs/点播模块.md 7.6
// 一个用户对一部电影只有一行（UNIQUE(user_id, movie_id)），上报即 upsert
type MovieProgress struct {
	ID        int64     `json:"id" gorm:"primaryKey"`                                  // 主键自增
	UserID    int64     `json:"userId" gorm:"uniqueIndex:idx_mprog;not null"`           // 用户
	MovieID   int64     `json:"movieId" gorm:"uniqueIndex:idx_mprog;index:idx_mprog_user;not null"` // 影片
	Position  int       `json:"position" gorm:"not null"`                               // 播放位置（秒）
	Duration  int       `json:"duration"`                                              // 影片时长（算进度百分比）
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime;index:idx_mprog_user"`   // 「最近观看」排序依据
}
