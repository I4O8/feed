package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// InsertLike 点赞写入一行
// 重复点赞撞 UNIQUE 唯一键 → 视为幂等成功，不报错
func InsertLike(like *model.Likes) error {
	err := DB.Create(like).Error
	if err == nil || errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}

// DeleteLike 取消点赞，删除一行
func DeleteLike(like *model.Likes) error {
	err := DB.Where("user_id = ? AND video_id = ?",
		like.UserID, like.VideoID).Delete(&model.Likes{}).Error
	if err != nil {
		return err
	}
	return nil
}

// CountLikes 统计某视频的点赞数（用于批量落库后重置 Redis 计数）
func CountLikes(videoID int64) (int64, error) {
	var count int64
	err := DB.Model(&model.Likes{}).Where("video_id = ?", videoID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
