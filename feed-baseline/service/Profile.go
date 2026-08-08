// Package service 业务逻辑层：承载业务规则、密码哈希、JWT 签发等核心逻辑
package service

import (
	"errors"

	"feed/dao"
	"feed/model"
)

var ErrUserNotFound = errors.New("用户不存在")

// GetProfile 根据用户 ID 查询公开资料
func GetProfile(userID int64) (*model.ProfileInfo, error) {
	acc, err := dao.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, ErrUserNotFound
	}
	profile := model.ProfileInfo{ID: acc.ID, Name: acc.Name, Avatar: acc.Avatar, Bio: acc.Bio, CreatedAt: acc.CreatedAt}
	return &profile, nil
}

// UpdateProfile 更新昵称和简介
func UpdateProfile(userID int64, name, bio *string) error {
	acc := &model.Account{ID: userID}
	if name != nil {
		acc.Name = *name
	}
	if bio != nil {
		acc.Bio = *bio
	}
	if err := dao.UpdateProfile(acc); err != nil {
		return err
	}
	return nil
}
