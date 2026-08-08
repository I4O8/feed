package model

import "time"

// Account 账号表结构（user 表）
type Account struct {
	ID        int64      `json:"id" gorm:"primaryKey"`                       // 用户 ID，主键，自增
	Name      string     `gorm:"not null;type:varchar(12)"`                  // 昵称（展示用，不参与登录，注册时默认生成"新用户_xxx"）
	Email     *string    `gorm:"unique;type:varchar(100)"`                   // 邮箱（登录凭证之一，可空，唯一；未填为 NULL）
	PNumber   *string    `gorm:"unique;type:varchar(16)"`                    // 手机号（登录凭证之一，可空，唯一；未填为 NULL）
	Password  string     `gorm:"type:varchar(255)"`                          // 密码哈希（bcrypt 存储，不存明文）
	Avatar    string     `gorm:"type:varchar(255)"`                          // 头像 URL（相对路径）
	Bio       string     `gorm:"type:text"`                                  // 个人简介
	Pdate     *time.Time                                                     // 会员到期时间（预留字段，暂不使用）
	Banned    bool       `gorm:"default:false"`                              // 是否封禁
	CreatedAt time.Time                                                      // 创建时间（GORM 自动维护）
	UpdatedAt time.Time                                                      // 更新时间（GORM 自动维护）
}

// ProfileInfo 用户公开资料（getProfile 响应，不含 Email/PNumber 等隐私字段）
type ProfileInfo struct {
	ID        int64     `json:"id"`        // 用户 ID
	Name      string    `json:"name"`      // 昵称
	Avatar    string    `json:"avatar"`    // 头像 URL
	Bio       string    `json:"bio"`       // 个人简介
	CreatedAt time.Time `json:"createdAt"` // 注册时间
}
