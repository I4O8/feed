package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"feed/dao"
)

// StartCountFlush 启动后台落库任务（main.go 启动时调用一次）
func StartCountFlush() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			flushDirtyCounts()
		}
	}()
}

// flushDirtyCounts 消费脏集合，把 Redis 计数覆盖写回 MySQL
func flushDirtyCounts() error {
	iter := dao.RDB.SScan(context.Background(), "video:dirty", 0, "", 100).Iterator()
	for iter.Next(context.Background()) {
		videoID := iter.Val()
		counts, err := dao.RDB.HGetAll(context.Background(), "video:count:"+videoID).Result()
		if err != nil {
			fmt.Printf("读取计数失败 video=%s: %v\n", videoID, err)
			continue
		}

		play, _ := strconv.ParseInt(counts["play"], 10, 64)
		like, _ := strconv.ParseInt(counts["like"], 10, 64)
		fav, _ := strconv.ParseInt(counts["fav"], 10, 64)
		id, _ := strconv.ParseInt(videoID, 10, 64)

		if err := dao.UpdateCounts(id, play, like, fav); err != nil {
			fmt.Printf("落库失败 video=%s: %v，保留脏标记\n", videoID, err)
			continue
		}
		if err := dao.RDB.SRem(context.Background(), "video:dirty", videoID).Err(); err != nil {
			fmt.Printf("SRem 失败 video=%s: %v\n", videoID, err)
		}

	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}
