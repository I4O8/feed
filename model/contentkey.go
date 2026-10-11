package model

import "time"

// ContentKey 内容密钥表：KeyValue 以信封加密（AES-GCM）落库，不存明文
// 文档：docs/点播模块.md 7.3 / 6.3
// A/B 两份分片共用同一把 content key（保证互换后仍能解密）
type ContentKey struct {
	ID          int64     `json:"id" gorm:"primaryKey"`                      // 主键自增
	KeyID       string    `json:"keyId" gorm:"not null;uniqueIndex;type:varchar(64)"` // key id（kid），manifest 中引用
	KeyValue    []byte    `json:"-" gorm:"not null;type:varbinary(255)"`     // AES-GCM 加密后的密钥（不落明文）
	Nonce       []byte    `json:"-" gorm:"not null;type:varbinary(32)"`      // GCM nonce
	Status      int8      `json:"status" gorm:"default:1"`                   // 1=启用 0=已轮换
	RotatedFrom int64     `json:"rotatedFrom"`                               // 轮换来源（预留：按内容位置分段换 key）
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`           // GORM 自动维护
}
