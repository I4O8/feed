package service

import (
	"feed/dao"
	"feed/model"
)

// Fav 收藏视频（退化版：MySQL favorites 表权威，计数同步更新 video.fav_count）
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
	return dao.IncrFavCount(videoID)
}

// UnFav 取消收藏（未收藏幂等）
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
	return dao.DecrFavCount(videoID)
}

// FavStatus 是否已收藏：查 MySQL 持久表
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

// FavList 我的收藏列表（退化版：逐条查视频 + 逐条查发布者，天然保持收藏记录顺序）
func FavList(userID, cursor int64, limit int) ([]*model.Video, int64, error) {
	favs, err := dao.ListFavorites(userID, cursor, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(favs) == 0 {
		return []*model.Video{}, 0, nil
	}

	videos := make([]*model.Video, 0, len(favs))
	for _, fav := range favs {
		v, err := dao.FindByVideoID(fav.VideoID)
		if err != nil {
			return nil, 0, err
		}
		if v == nil {
			continue // 视频被物理删除：自动跳过
		}
		account, err := dao.FindByID(v.Uploader)
		if err != nil {
			return nil, 0, err
		}
		if account == nil {
			v.Publisher = model.PublisherInfo{Name: "已注销用户", Avatar: ""}
		} else {
			v.Publisher = model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
		}
		videos = append(videos, v)
	}

	var nextCursor int64
	if len(favs) == limit {
		nextCursor = favs[len(favs)-1].ID
	}
	return videos, nextCursor, nil
}
