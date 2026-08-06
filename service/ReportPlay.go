package service

import (
	"context"
	"strconv"
	"time"

	"feed/dao"
)

// ReportPlay 播放上报：先确认视频存在，再播放量 +1；不存在返回 ErrVideoNotFound
func ReportPlay(videoID int64, deviceID string) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	// 去重Key
	key := "video:play:" + deviceID + ":" + strconv.FormatInt(videoID, 10)
	ok, err := dao.RDB.SetNX(context.Background(), key, "1", 24*time.Hour).Result()
	if err != nil {
		return err
	}
	if !ok { // 如果这个建已存在 那么返回nil 不++
		return nil
	}
	// 计数 +1（Redis 权威计数）
	if _, err := dao.RDB.HIncrBy(context.Background(), "video:count:"+strconv.FormatInt(videoID, 10), "play", 1).Result(); err != nil {
		return err
	}

	// 打脏标记：登记该视频计数被改过，待定时任务落库
	if err := dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err(); err != nil {
		return err
	}

	return nil
}
