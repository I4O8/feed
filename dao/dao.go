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
	db, err := gorm.Open(mysql.Open(config.Dsn), &gorm.Config{NamingStrategy: schema.NamingStrategy{
		SingularTable: true,
	}})
	if err != nil {
		panic(err)
	}
	DB = db
	if err := DB.AutoMigrate(&model.Account{}); err != nil {
		panic("User表创建失败" + err.Error())
	}
}

func initVideoTable() {
	if DB == nil {
		panic("DB未初始化，请先调用initUserTable()")
	}
	if err := DB.AutoMigrate(&model.Video{}); err != nil {
		panic("Video表创建失败" + err.Error())
	}
}

func InitSQL() {
	initUserTable()
	initVideoTable()
}
