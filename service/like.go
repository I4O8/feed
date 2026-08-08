package service

import (
	"context"
	"strconv"
	"time"

	"feed/dao"
)

// Like 点赞（乐观成功）Redis 短暂不可用时静默返回 nil，不报错
// 点赞是轻互动，数量允许轻微偏差，最终与sql一致，UI 反馈优先
func Like(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	key := "like:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(userID, 10)
	ok, _ := dao.RDB.SetNX(context.Background(), key,
		1, 7*24*time.Hour).Result()
	if ok {
		// 乐观成功：忽略错误；计数 +1 后打脏标记，由定时任务落库
		_ = dao.RDB.HIncrBy(context.Background(), "video:count:"+
			strconv.FormatInt(videoID, 10), "like", 1).Err()
		_ = dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err()
	}
	return nil
}

// UnLike 取消点赞
func UnLike(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}
	key := "like:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(userID, 10)
	n, _ := dao.RDB.Del(context.Background(), key).Result()
	if n == 0 {
		return nil
	}
	_ = dao.RDB.HIncrBy(context.Background(), "video:count:"+
		strconv.FormatInt(videoID, 10), "like", -1).Err()
	// 乐观成功：忽略错误；两个标记各自独立，unlike 通知删 likes 表，dirty 通知计数落库
	_ = dao.RDB.SAdd(context.Background(),
		"unlike:"+strconv.FormatInt(videoID, 10), userID).Err()
	_ = dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err()
	return nil
}

func LikeStatus(userID, videoID int64) (bool, error) {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return false, err
	}
	if video == nil {
		return false, ErrVideoNotFound
	}

	key := "like:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(userID, 10)
	n, _ := dao.RDB.Exists(context.Background(), key).Result()
	return n > 0, nil
}
