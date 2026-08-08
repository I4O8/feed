package service

import (
	"context"
	"strconv"
	"time"

	"feed/dao"
	"feed/model"

	"github.com/redis/go-redis/v9"
)

// markFanout per-粉丝幂等标记：SETNX 成功=true 表示"这个粉丝还没推过，可以 push"
// 正常完成不要清除（靠 TTL 1 天挡住重复投递）；只有 LPUSH 失败才 unmarkFanout 回滚
func markFanout(videoID, userID int64) bool {
	key := "fanout:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(userID, 10)
	ok, _ := dao.RDB.SetNX(context.Background(), key, 1, 24*time.Hour).Result()
	return ok
}

// unmarkFanout 失败回滚：LPUSH 失败时 DEL 幂等键，让 requeue 后能重试这个粉丝
func unmarkFanout(videoID, userID int64) error {
	key := "fanout:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(userID, 10)
	return dao.RDB.Del(context.Background(), key).Err()
}

// Feed 基于redis的feed流 组装部分与list.go无异
func Feed(userID, offset int64, count int) ([]*model.Video, int64, error) {
	res, err := dao.RDB.LRange(context.Background(), "feed:"+
		strconv.FormatInt(userID, 10), offset, offset+int64(count)-1).Result()
	if err != nil {
		return nil, 0, err
	}
	if len(res) == 0 {
		return []*model.Video{}, 0, nil
	}
	vIDs := make([]int64, len(res))
	for i := range res {
		vIDs[i], _ = strconv.ParseInt(res[i], 10, 64)
	}

	videos, err := dao.FindByVideoIDs(vIDs)
	if err != nil {
		return nil, -1, err
	}

	idMap := make(map[int64]struct{})
	for _, v := range videos {
		if v != nil {
			idMap[v.Uploader] = struct{}{}
		}
	}
	uIDs := make([]int64, 0, len(idMap))
	for id := range idMap {
		uIDs = append(uIDs, id)
	}

	accounts, err := dao.FindByIDs(uIDs)
	if err != nil {
		return nil, -1, err
	}

	accountMap := make(map[int64]model.Account)
	for i := range accounts {
		accountMap[accounts[i].ID] = accounts[i]
	}

	for i := range videos {
		if account, ok := accountMap[videos[i].Uploader]; !ok {
			publisherInfo := model.PublisherInfo{Name: "已注销用户", Avatar: ""}
			videos[i].Publisher = publisherInfo
		} else {
			publisherInfo := model.PublisherInfo{ID: account.ID, Name: account.Name, Avatar: account.Avatar}
			videos[i].Publisher = publisherInfo
		}
	}

	listCount(videos)
	var nextCursor int64
	if len(vIDs) == count {
		nextCursor = offset + int64(count)
	}

	return videos, nextCursor, nil
}

// fanout 视频入库后把 video_id 推给上传者的所有粉丝（同步版，MQ 阶段改异步）
func fanout(videoID, uploaderID int64) error {
	fans, err := dao.ListFollowers(uploaderID)
	if err != nil {
		return err
	}
	if len(fans) == 0 {
		return nil
	}

	// 分两次pipe 第一次setnx幂等健
	pipe1 := dao.RDB.Pipeline()
	setnxCmds := make([]*redis.BoolCmd, len(fans))
	for i, fanID := range fans {
		key := "fanout:" + strconv.FormatInt(videoID, 10) + ":" + strconv.FormatInt(fanID, 10)
		setnxCmds[i] = pipe1.SetNX(context.Background(), key, 1, 24*time.Hour)
	}
	if _, err := pipe1.Exec(context.Background()); err != nil {
		return err
	}
	// 对于返回ok的进行feed的扩充
	pipe2 := dao.RDB.Pipeline()
	for i, fanID := range fans {
		if !setnxCmds[i].Val() {
			continue // 已推过，跳过
		}
		feedKey := "feed:" + strconv.FormatInt(fanID, 10)
		val := strconv.FormatInt(videoID, 10)
		pipe2.LPush(context.Background(), feedKey, val)
		pipe2.LTrim(context.Background(), feedKey, 0, 199)
	}
	_, err = pipe2.Exec(context.Background())
	return err
}
