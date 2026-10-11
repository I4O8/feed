package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"feed/dao"
)

// 点播模块：电影播放量定时落库
// 与 video 的 flushCounts 同一套路：Redis 计数权威 + 脏集合定时落库（不重造）

// StartMovieCountFlush 启动电影计数落库任务（main.go 启动时调用一次）
func StartMovieCountFlush() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			flushMovieCounts()
		}
	}()
}

// flushMovieCounts 消费 movie:dirty 脏集合，把播放数覆盖写回 MySQL
func flushMovieCounts() error {
	iter := dao.RDB.SScan(context.Background(), "movie:dirty", 0, "", 100).Iterator()
	for iter.Next(context.Background()) {
		idStr := iter.Val()
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		play, err := dao.RDB.HGet(context.Background(), movieCountPrefix+idStr, "play").Int64()
		if err != nil {
			continue
		}
		if err := dao.UpdateMoviePlayCount(id, play); err != nil {
			fmt.Printf("电影播放量落库失败 id=%s: %v，保留脏标记\n", idStr, err)
			continue
		}
		dao.RDB.SRem(context.Background(), "movie:dirty", idStr)
	}
	return iter.Err()
}
