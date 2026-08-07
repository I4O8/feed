package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// InsertFavorite 收藏写入一行
// 重复收藏撞 UNIQUE 唯一键 → 视为幂等成功，不报错
func InsertFavorite(fav *model.Favorites) error {
	err := DB.Create(fav).Error
	if err == nil || errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}

// DeleteFavorite 取消收藏，删除一行
func DeleteFavorite(fav *model.Favorites) error {
	err := DB.Where("user_id = ? AND video_id = ?",
		fav.UserID, fav.VideoID).Delete(&model.Favorites{}).Error
	if err != nil {
		return err
	}
	return nil
}

// FavoriteExists 是否已收藏：三态返回，查不到 → (false, nil)
func FavoriteExists(userID, videoID int64) (bool, error) {
	var count int64
	err := DB.Model(&model.Favorites{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListFavorites 我的收藏列表：按 favorites.id 降序游标分页（cursor 为上一页最后一条 id）
func ListFavorites(userID, cursor int64, limit int) ([]*model.Favorites, error) {
	list := make([]*model.Favorites, 0)
	query := DB.Model(&model.Favorites{}).Where("user_id = ?", userID)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if err := query.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
