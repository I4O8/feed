package dao

import (
	"context"

	"feed/config"

	"github.com/redis/go-redis/v9"
)

// RDB Redis 客户端，供 service 层使用
var RDB *redis.Client

// InitRedis 初始化 Redis 连接并验证连通性
func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: "", // 本项目 Redis 无密码
		DB:       0,  // 默认库 0
	})
	if err := RDB.Ping(context.Background()).Err(); err != nil {
		panic("Redis 连接失败: " + err.Error())
	}
}
