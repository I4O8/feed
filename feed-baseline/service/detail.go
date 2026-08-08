package service

import (
	"errors"

	"feed/dao"
	"feed/model"
)

var ErrVideoNotFound = errors.New("video not found")

// GetDetail 传入视频ID，返回info结构体,err
// 退化版：直查 MySQL 组装，无缓存、无分布式锁、计数读 video 列
func GetDetail(videoID int64) (*model.Video, error) {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, ErrVideoNotFound
	}

	account, err := dao.FindByID(video.Uploader)
	if err != nil {
		return nil, err
	}
	if account == nil {
		video.Publisher = model.PublisherInfo{Name: "已注销的用户", Avatar: ""}
	} else {
		video.Publisher = model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
	}
	return video, nil
}
