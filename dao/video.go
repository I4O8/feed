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

// FindByVideoID 根据视频id查询，返回*结构体
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

// ListVideos 指定发布者 翻页 单页数量
func ListVideos(uploaderID, cursor int64, limit int) ([]*model.Video, error) {
	videoList := make([]*model.Video, 0)
	query := DB.Model(&model.Video{})
	if uploaderID > 0 {
		query = query.Where("uploader = ?", uploaderID)
	}
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if err := query.Order("id DESC").Limit(limit).Find(&videoList).Error; err != nil {
		return nil, err
	}
	return videoList, nil
}

// DeleteVideo 根据 ID 物理删除视频
func DeleteVideo(id int64) error {
	if err := DB.Delete(&model.Video{}, id).Error; err != nil {
		return err
	}
	return nil
}

// ReportPlay 播放量++ 原子操作 返回err
func ReportPlay(videoid int64) error {
	if err := DB.Model(&model.Video{}).Where("id = ?", videoid).Update("play_count", gorm.Expr("play_count + ?", 1)).Error; err != nil {
		return err
	}
	return nil
}
