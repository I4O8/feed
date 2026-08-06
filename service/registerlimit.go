package service

import (
	"context"
	"errors"
	"time"

	"feed/dao"
)

// ErrRatelimited 注册过于频繁（handler 映射 42900）
var ErrRatelimited = errors.New("too many register request")

// registerRateLimit 注册限流：固定窗口计数，一个 IP 一小时最多 5 次
func registerRateLimit(ctx context.Context, ip string) error {
	key := "reg:ip:" + ip

	// INCR：key 不存在自动初始化为 0 再 +1（第一次返回 1）
	n, err := dao.RDB.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	// 只有第一次才设 TTL，否则窗口永不重置
	if n == 1 {
		if _, err := dao.RDB.Expire(ctx, key, time.Hour).Result(); err != nil {
			return err
		}
	}
	// 超过阈值拒绝
	if n > 5 {
		return ErrRatelimited
	}
	return nil
}
