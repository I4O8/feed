package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// FindByID ID查找
func FindByID(id int64) (*model.Account, error) {
	var account model.Account
	if err := DB.First(&account, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// FindByEmail 邮箱查重
func FindByEmail(email string) (*model.Account, error) {
	var account model.Account
	if err := DB.Where("email = ?", email).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// FindByPNumber 手机号查重
func FindByPNumber(pNumber string) (*model.Account, error) {
	var account model.Account
	if err := DB.Where("p_number = ?", pNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// CreateAccount 创建账号
func CreateAccount(account *model.Account) (int64, error) {
	err := DB.Create(account).Error
	if err != nil {
		return -1, err
	}
	return int64(account.ID), nil
}

// UpdateProfile 期望传入含id,name,bio的account指针
func UpdateProfile(account *model.Account) error {
	if err := DB.Updates(account).Error; err != nil {
		return err
	}
	return nil
}

// GetPassword 输入userID(默认一定存在),返回密码哈希，err
func GetPassword(userID int64) (string, error) {
	var hash string
	if err := DB.Model(&model.Account{}).Where("id = ?", userID).Pluck("password", &hash).Error; err != nil {
		return "", err
	}
	return hash, nil
}

// UpdatePassword 输入userID,新密码哈希
func UpdatePassword(userID int64, newHash string) error {
	if err := DB.Model(&model.Account{}).Where("id = ?", userID).Update("password", newHash).Error; err != nil {
		return err
	}
	return nil
}

// UpdateAvatar 传入userID，头像路径，返回err
func UpdateAvatar(userID int64, path string) error {
	if err := DB.Model(&model.Account{}).Where("id = ?", userID).Update("avatar", path).Error; err != nil {
		return err
	}
	return nil
}

// FindByIDs 根据多个id进行单次查询 避免N+1
func FindByIDs(ids []int64) ([]model.Account, error) {
	var accounts []model.Account
	if err := DB.Where("id in ?", ids).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}
