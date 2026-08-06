package service

import (
	"context"
	"strconv"

	"feed/dao"
	"feed/model"

	"github.com/redis/go-redis/v9"
)

func listCount(videos []*model.Video) {
	pipe := dao.RDB.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(videos))
	for i := range videos {
		cmds[i] = pipe.HGetAll(context.Background(), "video:count:"+strconv.FormatInt(videos[i].ID, 10))
	}
	pipe.Exec(context.Background())

	for i := range cmds {
		res, _ := cmds[i].Result()
		if play, ok := res["play"]; ok {
			videos[i].PlayCount, _ = strconv.ParseInt(play, 10, 64)
		}
		if like, ok := res["like"]; ok {
			videos[i].LikeCount, _ = strconv.ParseInt(like, 10, 64)
		}
		if fav, ok := res["fav"]; ok {
			videos[i].FavCount, _ = strconv.ParseInt(fav, 10, 64)
		}

	}
}

// List 视频列表：游标分页，可选按发布者过滤（uploaderID 为 0 = 全站）
// 返回已组装 Publisher 的视频切片、下一页游标（0 = 没有更多）
func List(uploaderID, cursor int64, limit int) ([]*model.Video, int64, error) {
	videos, err := dao.ListVideos(uploaderID, cursor, limit)
	if err != nil {
		return nil, -1, err
	}

	// 收集并去重发布者 id（map 当集合，key 天然不重复）
	idMap := make(map[int64]struct{})
	for _, v := range videos {
		if v != nil {
			idMap[v.Uploader] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}

	// 批量查回所有发布者，一次 IN 查询，避免 N+1
	accounts, err := dao.FindByIDs(ids)
	if err != nil {
		return nil, -1, err
	}

	// 建查找表：id → 账户
	accountMap := make(map[int64]model.Account)
	for i := range accounts {
		accountMap[accounts[i].ID] = accounts[i]
	}

	// 遍历视频，用 Uploader 查表填 Publisher；查不到（发布者被删）兜底为已注销
	for i := range videos {
		if account, ok := accountMap[videos[i].Uploader]; !ok {
			publisherInfo := model.PublisherInfo{Name: "已注销用户", Avatar: ""}
			videos[i].Publisher = publisherInfo
		} else {
			publisherInfo := model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
			videos[i].Publisher = publisherInfo
		}
	}

	listCount(videos)

	// 取满 limit 才说明还有更旧的，nextCursor = 最后一条 id；没取满说明到底，nextCursor = 0
	var nextCursor int64
	if len(videos) == limit {
		nextCursor = videos[len(videos)-1].ID
	}
	return videos, nextCursor, nil
}
