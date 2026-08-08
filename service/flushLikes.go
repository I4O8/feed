package service

import (
	"context"
	"time"

	"feed/dao"
)

func StartLikeFlush() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			flushLikes()
		}
	}()
}

func flushLikes() error {
	iter := dao.RDB.SScan(context.Background(), "like:video:*", 0, "", 0).Iterator()
	var vIDs int64
	for iter.Next(context.Background()) {
		videoID := iter.Val()
	}
}
