package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// CreateContentKey 写入一把内容密钥（KeyValue 已是信封加密后的密文）
func CreateContentKey(k *model.ContentKey) (int64, error) {
	if err := DB.Create(k).Error; err != nil {
		return -1, err
	}
	return k.ID, nil
}

// FindContentKeyByID 按主键取密钥
func FindContentKeyByID(id int64) (*model.ContentKey, error) {
	var k *model.ContentKey
	if err := DB.First(&k, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return k, nil
}

// FindContentKeyByKeyID 按 kid 取密钥（license 端点按 manifest 里的 kid 反查）
func FindContentKeyByKeyID(keyID string) (*model.ContentKey, error) {
	var k *model.ContentKey
	if err := DB.Where("key_id = ?", keyID).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return k, nil
}
