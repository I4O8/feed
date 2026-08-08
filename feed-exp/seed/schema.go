package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// tablesDDL 六张表 DDL（与 GORM AutoMigrate 生成的表结构兼容，`describe` 为 MySQL 保留字需反引号）
var tablesDDL = []string{
	`CREATE TABLE IF NOT EXISTS account (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(12) NOT NULL,
		email VARCHAR(100) UNIQUE,
		p_number VARCHAR(16) UNIQUE,
		password VARCHAR(255) NOT NULL,
		avatar VARCHAR(255) DEFAULT '',
		bio TEXT,
		pdate DATETIME NULL,
		banned BOOLEAN DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS video (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		cover_url VARCHAR(255),
		video_url VARCHAR(255) NOT NULL,
		` + "`describe`" + ` TEXT,
		uploader BIGINT NOT NULL,
		duration INT,
		play_count BIGINT DEFAULT 0,
		like_count BIGINT DEFAULT 0,
		fav_count BIGINT DEFAULT 0,
		status TINYINT DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS follows (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		follower_id BIGINT NOT NULL,
		followee_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE KEY idx_follow (follower_id, followee_id),
		KEY idx_followee (followee_id)
	)`,
	`CREATE TABLE IF NOT EXISTS likes (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		video_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE KEY idx_like (user_id, video_id),
		KEY idx_like_video (video_id)
	)`,
	`CREATE TABLE IF NOT EXISTS favorites (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		video_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE KEY idx_fav (user_id, video_id),
		KEY idx_fav_user (user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS feed (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		video_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL,
		KEY idx_uid_id (user_id, id)
	)`,
}

// connectMySQL 连接 MySQL（seed 走原生 SQL，不用 GORM）
func connectMySQL(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL 连接失败: %w", err)
	}
	return db, nil
}

// connectRedis 连接 Redis
func connectRedis(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}
	return rdb, nil
}

// createTables 建表（IF NOT EXISTS，幂等）
func createTables(db *sql.DB) error {
	for _, ddl := range tablesDDL {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("建表失败: %w\nDDL: %s", err, ddl)
		}
	}
	return nil
}

// truncateTables 清空六表（重置自增）
func truncateTables(db *sql.DB) error {
	tables := []string{"feed", "favorites", "likes", "follows", "video", "account"}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}
	defer db.Exec("SET FOREIGN_KEY_CHECKS=1")
	for _, t := range tables {
		if _, err := db.Exec("TRUNCATE TABLE " + t); err != nil {
			return fmt.Errorf("TRUNCATE %s 失败: %w", t, err)
		}
	}
	return nil
}
