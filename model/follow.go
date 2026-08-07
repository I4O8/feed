package model

import "time"

type Follows struct {
	ID         int64 `json:"id" gorm:"primaryKey"`                                                 // 主键自增
	FollowerID int64 `json:"followerId" gorm:"uniqueIndex:idx_follow;not null"`                    // 谁关注（主动方）
	FolloweeID int64 `json:"followeeId" gorm:"uniqueIndex:idx_follow;index:idx_followee;not null"` // 关注谁（被动方）
	CreatedAt  time.Time
}

// UserBrief 关注/粉丝列表的单项 · 文档：docs/feed模块.md 第六节接口 3/4
type UserBrief struct {
	ID     int64  `json:"id"`     // 用户 ID
	Name   string `json:"name"`   // 昵称
	Avatar string `json:"avatar"` // 头像 URL
}
