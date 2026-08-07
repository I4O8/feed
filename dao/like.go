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
