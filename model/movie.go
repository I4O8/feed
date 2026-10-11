package model

import "time"

// 电影上架状态
const (
	MovieStatusPacking = 0 // 转码打包中
	MovieStatusReady   = 1 // 就绪，可播放
	MovieStatusFailed  = 2 // 打包失败
	MovieStatusOffline = 3 // 已下架
)

// Movie 电影元数据表，对应 movie 表；点播模块片库的主体
// 文档：docs/点播模块.md 7.1
// 基础字段（Title/Year/Rating/Region）做成列 → 可排序可筛选；
// 复杂主观字段（Genres/Cast/Crew）存 JSON → 导入直接落库，简单
type Movie struct {
	ID               int64     `json:"id" gorm:"primaryKey"`                            // 主键自增
	Title            string    `json:"title" gorm:"not null;type:varchar(255)"`         // 片名
	OriginalTitle    string    `json:"originalTitle" gorm:"type:varchar(255)"`          // 原名
	Description      string    `json:"description" gorm:"type:text"`                    // 简介
	PosterURL        string    `json:"posterUrl" gorm:"type:varchar(255)"`              // 海报（本地化，见 7.7）
	BackdropURL      string    `json:"backdropUrl" gorm:"type:varchar(255)"`            // 背景图
	Duration         int       `json:"duration"`                                        // 时长（秒）
	Year             int       `json:"year"`                                            // 上映年份
	Region           string    `json:"region" gorm:"type:varchar(64)"`                  // 地区
	Rating           float64   `json:"rating" gorm:"type:decimal(3,1)"`                 // 评分
	Genres           GenreList `json:"genres" gorm:"type:json"`                         // 类型（数组）
	Cast             CastList  `json:"cast" gorm:"type:json"`                           // 演员（含角色）
	Crew             CrewList  `json:"crew" gorm:"type:json"`                           // 导演 / 编剧等（含职务）
	TmdbID           int64     `json:"tmdbId" gorm:"uniqueIndex;default:0"`             // 外部数据源 ID（导入用，0=未知）
	SourcePathA      string    `json:"-" gorm:"type:varchar(255)"`                      // 母版 A 磁盘路径（A/B 水印用）
	SourcePathB      string    `json:"-" gorm:"type:varchar(255)"`                      // 母版 B 磁盘路径（与 A 强制关键帧对齐）
	SegmentCount     int       `json:"segmentCount"`                                    // 逐片分析得出的分片/IDR 段数（A/B 一致）
	SegmentDurations FloatList `json:"segmentDurations" gorm:"type:json"`               // 逐段时长（秒，浮点）数组
	PlayCount        int64     `json:"playCount" gorm:"default:0"`                      // 播放次数（Redis 权威，定时落库）
	Status           int8      `json:"status" gorm:"default:0"`                         // 0=转码中 1=就绪 2=失败 3=下架
	FailReason       string    `json:"failReason" gorm:"type:varchar(255)"`             // 打包失败原因
	CreatedAt        time.Time `json:"createdAt" gorm:"autoCreateTime"`                 // GORM 自动维护
	UpdatedAt        time.Time `json:"-" gorm:"autoUpdateTime"`                         // GORM 自动维护
	// 以下为响应字段，非数据库列
	Favored  bool `json:"favored" gorm:"-"`  // 当前用户是否收藏（登录时填）
	Position int  `json:"position" gorm:"-"` // 当前用户观看进度秒（登录时填，续播用）
}
