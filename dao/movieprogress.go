package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpsertMovieProgress 上报进度：一个用户一部电影一行，冲突则覆盖 position/duration
func UpsertMovieProgress(p *model.MovieProgress) error {
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "movie_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"position", "duration"}),
	}).Create(p).Error
}

// FindMovieProgress 取某用户对某电影的进度（续播定位）；查不到返回 (nil, nil)
func FindMovieProgress(userID, movieID int64) (*model.MovieProgress, error) {
	var p *model.MovieProgress
	if err := DB.Where("user_id = ? AND movie_id = ?", userID, movieID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// ListMovieProgress 观看历史：按 updated_at 降序游标分页
// cursor 传上一页最后一条的 updated_at 的 Unix 秒；0 = 第一页
func ListMovieProgress(userID, cursor int64, limit int) ([]*model.MovieProgress, error) {
	list := make([]*model.MovieProgress, 0)
	q := DB.Model(&model.MovieProgress{}).Where("user_id = ?", userID)
	if cursor > 0 {
		q = q.Where("updated_at < FROM_UNIXTIME(?)", cursor)
	}
	if err := q.Order("updated_at DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
