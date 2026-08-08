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

// DeleteLike 取消点赞，删除一行；返回是否真删了（RowsAffected > 0）
func DeleteLike(like *model.Likes) (bool, error) {
	res := DB.Where("user_id = ? AND video_id = ?",
		like.UserID, like.VideoID).Delete(&model.Likes{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// LikeExists 是否已点赞：三态返回，查不到 → (false, nil)
func LikeExists(userID, videoID int64) (bool, error) {
	var count int64
	err := DB.Model(&model.Likes{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
