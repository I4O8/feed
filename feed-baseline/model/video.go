package model

import "time"

// Video 数据库模型，对应 video 表，兼作列表/详情接口的响应结构
type Video struct {
	ID        int64         `json:"id" gorm:"primaryKey"`                        // 主键ID
	Title     string        `json:"title" gorm:"not null"`                       // 视频标题
	CoverURL  string        `json:"coverUrl" gorm:"type:varchar(255)"`           // 封面图URL，可空
	VideoURL  string        `json:"videoUrl" gorm:"not null;type:varchar(255)"`  // 视频文件URL
	Describe  string        `json:"describe" gorm:"type:text"`                   // 视频描述，可空
	Uploader  int64         `json:"-" gorm:"not null"`                           // 发布者用户ID（不出现在响应，由 publisher 提供）
	Duration  int           `json:"duration"`                                    // 视频时长（秒）
	PlayCount int64         `json:"playCount" gorm:"default:0"`                  // 播放次数
	LikeCount int64         `json:"likeCount" gorm:"default:0"`                  // 点赞数
	FavCount  int64         `json:"favCount" gorm:"default:0"`                   // 收藏数
	Status    int8          `json:"status" gorm:"default:1"`                     // 状态：1上架，0下架
	Publisher PublisherInfo `json:"publisher" gorm:"-"`                          // 发布者信息（响应字段，非数据库列）
	CreatedAt time.Time     `json:"createdAt" gorm:"autoCreateTime"`             // 创建时间，GORM自动维护
	UpdatedAt time.Time     `json:"-" gorm:"autoUpdateTime"`                     // 更新时间（数据库列，响应不输出）
}

// PublisherInfo 发布者信息（嵌套在 Video 中）
type PublisherInfo struct {
	Name   string `json:"name"`   // 发布者昵称
	Avatar string `json:"avatar"` // 发布者头像URL
}
