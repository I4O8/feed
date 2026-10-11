package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// InsertMovieFavorite 收藏写入一行；重复收藏撞 UNIQUE → 幂等成功
func InsertMovieFavorite(fav *model.MovieFavorite) error {
	err := DB.Create(fav).Error
	if err == nil || errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}

// DeleteMovieFavorite 取消收藏
func DeleteMovieFavorite(userID, movieID int64) error {
	return DB.Where("user_id = ? AND movie_id = ?", userID, movieID).
		Delete(&model.MovieFavorite{}).Error
}

// MovieFavoriteExists 是否已收藏
func MovieFavoriteExists(userID, movieID int64) (bool, error) {
	var n int64
	err := DB.Model(&model.MovieFavorite{}).
		Where("user_id = ? AND movie_id = ?", userID, movieID).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListMovieFavorites 我的电影收藏：按 id 降序游标分页
func ListMovieFavorites(userID, cursor int64, limit int) ([]*model.MovieFavorite, error) {
	list := make([]*model.MovieFavorite, 0)
	q := DB.Model(&model.MovieFavorite{}).Where("user_id = ?", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	if err := q.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
