package dao

import (
	"time"

	"gorm.io/gorm"
)

// 连接池配置：优化版 / 退化版必须完全一致，保证压测对比公平（见 docs/性能对比实验.md §3.4）
const (
	PoolMaxOpenConns    = 200
	PoolMaxIdleConns    = 50
	PoolConnMaxLifetime = 30 * time.Minute
)

// setupPool 显式设置连接池参数（gorm.Open 默认不设上限，高并发下连接数无界增长）
func setupPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(PoolMaxOpenConns)
	sqlDB.SetMaxIdleConns(PoolMaxIdleConns)
	sqlDB.SetConnMaxLifetime(PoolConnMaxLifetime)
	return nil
}
