package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// InsertFollow   关注写入一行（followerID 关注 followeeID）
// 重复关注撞 UNIQUE 唯一键 → 视为幂等成功，不报错
func InsertFollow(follow *model.Follows) error {
	err := DB.Create(follow).Error
	if err == nil || errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil
	}
	return err
}

// DeleteFollow   取关删除一行
func DeleteFollow(follow *model.Follows) error {
	err := DB.Where("follower_id = ? AND followee_id = ?",
		follow.FollowerID, follow.FolloweeID).Delete(&model.Follows{}).Error
	if err != nil {
		return err
	}
	return nil
}

// ListFollows    我关注了谁：WHERE follower_id = ?
func ListFollows(follwerID int64) ([]int64, error) {
	var follows []*model.Follows
	if err := DB.Where("follower_id = ?", follwerID).Find(&follows).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(follows))
	for i := range follows {
		ids = append(ids, follows[i].FolloweeID)
	}
	return ids, nil
}

// ListFollowers  谁关注了我：WHERE followee_id = ?
func ListFollowers(followeeID int64) ([]int64, error) {
	var follows []*model.Follows
	if err := DB.Where("followee_id = ?", followeeID).Find(&follows).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(follows))
	for i := range follows {
		ids = append(ids, follows[i].FollowerID)
	}
	return ids, nil
}
