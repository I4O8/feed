package service

import (
	"context"
	"strconv"
	"time"

	"feed/dao"
)

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
		// 我不关心标记设置成功与否，但是为了防止panic,我接受了err值
		_, err = dao.RDB.Incr(context.Background(), "video:likecount"+strconv.FormatInt(videoID, 10)).Result()
		if err == nil {
			_ = dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err()
		}
		return
	}
	return nil
}


Func UnLike()
Fnuc LikeStatus()
