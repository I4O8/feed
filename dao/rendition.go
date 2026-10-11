package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// CreateRendition 新建一条产物记录
func CreateRendition(r *model.MovieRendition) (int64, error) {
	if err := DB.Create(r).Error; err != nil {
		return -1, err
	}
	return r.ID, nil
}

// FindRenditionsByMovie 列出某电影的全部产物
func FindRenditionsByMovie(movieID int64) ([]*model.MovieRendition, error) {
	list := make([]*model.MovieRendition, 0)
	if err := DB.Where("movie_id = ?", movieID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// FindRendition 按「电影 + 封装 + 加密方案」取产物（闸门按客户端能力选片用）
func FindRendition(movieID int64, container, drmScheme string) (*model.MovieRendition, error) {
	var r *model.MovieRendition
	q := DB.Where("movie_id = ?", movieID)
	if container != "" {
		q = q.Where("container = ?", container)
	}
	if drmScheme != "" {
		q = q.Where("drm_scheme = ?", drmScheme)
	}
	if err := q.Order("bitrate DESC").First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

// FindRenditionByID 按 ID 取产物
func FindRenditionByID(id int64) (*model.MovieRendition, error) {
	var r *model.MovieRendition
	if err := DB.First(&r, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

// DeleteRenditionsByMovie 重打包前清掉旧产物
func DeleteRenditionsByMovie(movieID int64) error {
	return DB.Where("movie_id = ?", movieID).Delete(&model.MovieRendition{}).Error
}
