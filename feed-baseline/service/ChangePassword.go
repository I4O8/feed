package service

import (
	"errors"

	"feed/dao"
)

var ErrOldPasswordWrong = errors.New("旧密码错误")

// ChangePassword 输入UserID，新密码，旧密码，返回err
func ChangePassword(userID int64, newPassword, oldPassword string) error {
	hash, err := dao.GetPassword(userID)
	if err != nil {
		return err
	}
	if VerifyPassword(hash, oldPassword) {
		newHash, err := HashPassword(newPassword)
		if err != nil {
			return err
		}
		if err := dao.UpdatePassword(userID, newHash); err != nil {
			return err
		}
		return nil
	}
	return ErrOldPasswordWrong
}
