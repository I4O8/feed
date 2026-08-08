package service

import (
	"context"
	"strconv"

	"feed/dao"
	"feed/model"
)

// Fav 收藏视频（严格错误处理）
// 收藏是持久数据，MySQL favorites 表为权威，任一 Redis/MySQL 操作失败都向上返回错误，不吞
func Fav(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	// 已收藏 → 幂等成功
	exists, err := dao.FavoriteExists(userID, videoID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 写库（UNIQUE 唯一键兜底并发重复收藏）
	if err := dao.InsertFavorite(&model.Favorites{UserID: userID, VideoID: videoID}); err != nil {
		return err
	}

	// 计数 +1 走现有计数模型（Redis hash，待定时任务落库）
	if _, err := dao.RDB.HIncrBy(context.Background(),
		"video:count:"+strconv.FormatInt(videoID, 10), "fav", 1).Result(); err != nil {
		return err
	}

	// 打脏标记：通知 flushCounts 该视频计数变了
	if err := dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err(); err != nil {
		return err
	}

	return nil
}

// UnFav 取消收藏（严格错误处理，未收藏幂等）
func UnFav(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	// 未收藏 → 幂等成功（不删库，也不动计数）
	exists, err := dao.FavoriteExists(userID, videoID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if err := dao.DeleteFavorite(&model.Favorites{UserID: userID, VideoID: videoID}); err != nil {
		return err
	}

	if _, err := dao.RDB.HIncrBy(context.Background(),
		"video:count:"+strconv.FormatInt(videoID, 10), "fav", -1).Result(); err != nil {
		return err
	}

	if err := dao.RDB.SAdd(context.Background(), "video:dirty", videoID).Err(); err != nil {
		return err
	}

	return nil
}

// FavStatus 是否已收藏（严格错误处理，查 MySQL 持久表）
func FavStatus(userID, videoID int64) (bool, error) {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return false, err
	}
	if video == nil {
		return false, ErrVideoNotFound
	}

	return dao.FavoriteExists(userID, videoID)
}

func FavList(userID, cursor int64, limit int) ([]*model.Video, int64, error) {
	favs, err := dao.ListFavorites(userID, cursor, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(favs) == 0 {
		return []*model.Video{}, 0, nil
	}
	vIDs := make([]int64, len(favs))
	for i := range favs {
		vIDs[i] = favs[i].VideoID
	}
	videos, err := dao.FindByVideoIDs(vIDs)
	if err != nil {
		return nil, 0, err
	}

	// 按 favs 顺序重排：IN 查询不保证顺序，以收藏记录 id 降序为准
	videoMap := make(map[int64]*model.Video, len(videos))
	for _, v := range videos {
		if v != nil {
			videoMap[v.ID] = v
		}
	}
	ordered := make([]*model.Video, 0, len(favs))
	for _, fav := range favs {
		if v, ok := videoMap[fav.VideoID]; ok {
			ordered = append(ordered, v)
		}
		// 视频被物理删除：查不到自动跳过（无脏数据）
	}

	idMap := make(map[int64]struct{})
	for _, v := range ordered {
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
		return nil, 0, err
	}

	accountMap := make(map[int64]model.Account)
	for i := range accounts {
		accountMap[accounts[i].ID] = accounts[i]
	}
	for i := range ordered {
		if account, ok := accountMap[ordered[i].Uploader]; !ok {
			publisherInfo := model.PublisherInfo{Name: "已注销用户", Avatar: ""}
			ordered[i].Publisher = publisherInfo
		} else {
			publisherInfo := model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
			ordered[i].Publisher = publisherInfo
		}
	}

	listCount(ordered)

	var nextCursor int64
	if len(favs) == limit {
		nextCursor = favs[len(favs)-1].ID
	}
	return ordered, nextCursor, nil
}
