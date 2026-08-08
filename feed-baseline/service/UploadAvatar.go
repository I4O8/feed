package service

import (
	"os"
	"path/filepath"

	"feed/config"
	"feed/dao"
)

// UploadAvatar 传入用户id，图片数据，图片名称 返回图片路径，err
func UploadAvatar(userID int64, pic []byte, fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	newFileName := generateRandomString(20) + ext

	savePath := filepath.Join(config.UploadDir, newFileName)
	if err := os.MkdirAll(config.UploadDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(savePath, pic, 0o644); err != nil {
		return "", err
	}

	urlPath := "/uploads/avatar/" + newFileName
	if err := dao.UpdateAvatar(userID, urlPath); err != nil {
		return "", err
	}

	return urlPath, nil
}
