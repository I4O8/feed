package dao

import (
	"feed/config"
	"feed/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// initUserTable 初始化USER表
func initUserTable() {
	db, err := gorm.Open(mysql.Open(config.Dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		TranslateError: true, // 唯一键冲突 → gorm.ErrDuplicatedKey
	})
	if err != nil {
		panic(err)
	}
	DB = db
	if err := setupPool(DB); err != nil { // 显式连接池，与退化版一致（压测公平）
		panic("连接池设置失败: " + err.Error())
	}
	if err := DB.AutoMigrate(&model.Account{}); err != nil {
		panic("User表创建失败" + err.Error())
	}
}

// initVideoTable 初始化视频表
func initVideoTable() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(&model.Video{}); err != nil {
		panic("Video表创建失败" + err.Error())
	}
}

// initFollowsTable 初始化关注表
func initFollowsTable() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(&model.Follows{}); err != nil {
		panic("Follows{}表创建失败" + err.Error())
	}
}

// initLikesTable 初始化点赞表
func initLikesTable() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(&model.Likes{}); err != nil {
		panic("Likes{}表创建失败" + err.Error())
	}
}

// initFavoritesTable 初始化收藏表
func initFavoritesTable() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(&model.Favorites{}); err != nil {
		panic("Favorites{}表创建失败" + err.Error())
	}
}

// initVodTables 初始化点播模块六表：电影 / 产物 / 内容密钥 / 播放会话 / 收藏 / 进度
// 文档：docs/点播模块.md 七、表设计
func initVodTables() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(
		&model.Movie{},
		&model.MovieRendition{},
		&model.ContentKey{},
		&model.PlaybackSession{},
		&model.MovieFavorite{},
		&model.MovieProgress{},
	); err != nil {
		panic("点播表创建失败" + err.Error())
	}
}

// InitSQL 初始化建表：账号/视频/关注/点赞/收藏 五表 + 点播六表
func InitSQL() {
	initUserTable()
	initVideoTable()
	initFollowsTable()
	initLikesTable()
	initFavoritesTable()
	initVodTables()
}
