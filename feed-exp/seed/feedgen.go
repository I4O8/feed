package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

// computeFeeds 为每个用户计算 feed：取其关注者的视频，按视频 id 降序取前 feedLimit 条
// 返回 map[userID][]videoID（降序，最新在前）
func computeFeeds(follows map[int64][]int64, byUploader map[int64][]int64, feedLimit, nUsers int) map[int64][]int64 {
	fmt.Printf("计算 %d 用户的 feed（每人上限 %d）...\n", nUsers, feedLimit)
	feeds := make(map[int64][]int64, nUsers)
	for u := int64(1); u <= int64(nUsers); u++ {
		var all []int64
		for _, f := range follows[u] {
			all = append(all, byUploader[f]...)
		}
		if len(all) == 0 {
			continue
		}
		// 降序：最新视频在前
		sort.Slice(all, func(i, j int) bool { return all[i] > all[j] })
		if len(all) > feedLimit {
			all = all[:feedLimit]
		}
		feeds[u] = all
	}
	return feeds
}

// writeFeedMySQL 写退化版 feed 表：每用户按视频 id 升序插入（最旧先插、最新后插）
// → 最新视频获得最大 feed.id → ORDER BY id DESC 即最新在前，与优化版列表顺序一致
func writeFeedMySQL(db *sql.DB, feeds map[int64][]int64, batch int, now time.Time) error {
	total := 0
	rows := make([][]any, 0, batch*10)
	base := "INSERT INTO feed (user_id, video_id, created_at) VALUES "
	for u, ids := range feeds {
		for i := len(ids) - 1; i >= 0; i-- { // ids 降序，反着=升序
			rows = append(rows, []any{u, ids[i], now})
		}
		total += len(ids)
		if len(rows) >= batch {
			if err := execBatch(db, base, batch, rows); err != nil {
				return err
			}
			rows = rows[:0]
		}
	}
	if len(rows) > 0 {
		if err := execBatch(db, base, batch, rows); err != nil {
			return err
		}
	}
	fmt.Printf("feed 表写入 %d 条\n", total)
	return nil
}

// writeFeedRedis 写优化版 Redis feed List：RPUSH 按降序（先最新）
// → list[0] = 最新，与优化版 fanout LPUSH 后的语义一致；LTRIM 限长 200
func writeFeedRedis(rdb *redis.Client, feeds map[int64][]int64) error {
	ctx := context.Background()
	pipe := rdb.Pipeline()
	count := 0
	for u, ids := range feeds {
		key := fmt.Sprintf("feed:%d", u)
		args := make([]any, len(ids))
		for i, id := range ids {
			args[i] = id
		}
		pipe.RPush(ctx, key, args...)
		pipe.LTrim(ctx, key, 0, 199)
		count++
		if count%500 == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("Redis feed 写入失败: %w", err)
			}
			pipe = rdb.Pipeline()
		}
	}
	if count%500 != 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("Redis feed 写入失败: %w", err)
		}
	}
	fmt.Printf("Redis feed 写入 %d 个用户\n", count)
	return nil
}

// writeCountsRedis 写优化版计数 hash：video:count:<id> 与 MySQL 列一致
// 保证两版读到的计数相同（公平）
func writeCountsRedis(rdb *redis.Client, infos []videoInfo, batch int) error {
	ctx := context.Background()
	pipe := rdb.Pipeline()
	for i, v := range infos {
		pipe.HSet(ctx, fmt.Sprintf("video:count:%d", v.id), "play", v.play, "like", v.like, "fav", v.fav)
		if (i+1)%batch == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("Redis 计数写入失败: %w", err)
			}
			pipe = rdb.Pipeline()
		}
	}
	if len(infos)%batch != 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("Redis 计数写入失败: %w", err)
		}
	}
	fmt.Printf("Redis 计数 hash 写入 %d 个视频\n", len(infos))
	return nil
}
