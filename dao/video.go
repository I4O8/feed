package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// CreateVideo 创建一条视频记录
func CreateVideo(video *model.Video) (int64, error) {
	err := DB.Create(video).Error
	if err != nil {
		return -1, err
	}
	return int64(video.ID), nil
}

func FindByVideoID(videoID int64) (*model.Video, error) {
	var tmp *model.Video
	if err := DB.First(&tmp, videoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return tmp, nil
}
