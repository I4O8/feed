package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"feed/dao"
	"feed/model"
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
	// 第一层：遍历 like:dirty 脏集合，拿到需要同步的视频 id
	iter := dao.RDB.SScan(context.Background(), "like:dirty", 0, "", 100).Iterator()
	for iter.Next(context.Background()) {
		videoIDStr := iter.Val()
		if videoIDStr == "" {
			continue
		}
		videoID, err := strconv.ParseInt(videoIDStr, 10, 64)
		if err != nil {
			return err
		}

		// 先删多余：处理 unlike 标记（先删后增，重赞用户会被状态键重新补齐）
		unLikedUsers, err := dao.RDB.SMembers(context.Background(), "unlike:"+videoIDStr).Result()
		if err != nil {
			return err
		}
		for _, uidStr := range unLikedUsers {
			userID, err := strconv.ParseInt(uidStr, 10, 64)
			if err != nil {
				return err
			}
			if err := dao.DeleteLike(&model.Likes{UserID: userID, VideoID: videoID}); err != nil {
				return err
			}
		}

		// 再按状态键补齐：以 Redis 状态键为权威，把当前真实点赞关系写回 likes 表
		var cursor uint64
		for {
			keys, next, err := dao.RDB.Scan(context.Background(), cursor,
				"like:"+videoIDStr+":*", 100).Result()
			if err != nil {
				return err
			}
			// 遍历这批 key，拆出 userId 后落库（InsertLike 已吞唯一键，幂等）
			for _, key := range keys {
				parts := strings.Split(key, ":") // ["like", "<videoID>", "<userId>"]
				if len(parts) != 3 {
					continue
				}
				uid, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return err
				}
				if err := dao.InsertLike(&model.Likes{UserID: uid, VideoID: videoID}); err != nil {
					return err
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}

		// 计数重置：以 MySQL likes 表 count(*) 为准，HSet 只覆盖 like 字段（不影响 play/fav）
		count, err := dao.CountLikes(videoID)
		if err != nil {
			return err
		}
		if err := dao.RDB.HSet(context.Background(), "video:count:"+videoIDStr,
			"like", count).Err(); err != nil {
			return err
		}

		// 同步完成：清理脏标记与 unlike 标记（失败保留，下次重试）
		if err := dao.RDB.SRem(context.Background(), "like:dirty", videoIDStr).Err(); err != nil {
			return err
		}
		if err := dao.RDB.Del(context.Background(), "unlike:"+videoIDStr).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}
