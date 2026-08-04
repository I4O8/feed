package service

import (
	"errors"

	"feed/dao"
	"feed/model"
)

var ErrVideoNotFound = errors.New("video not found")

// GetDetail 传入视频ID，返回info结构体,err
func GetDetail(videoID int64) (*model.Video, error) {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, ErrVideoNotFound
	}

	var name, avatar string

	account, err := dao.FindByID(video.Uploader)
	if err != nil {
		return nil, err
	}
	if account == nil {
		name = "已注销的用户"
		avatar = "" // 约定：avatar 未设置为空字符串，前端显示占位
	} else {
		name = account.Name
		avatar = account.Avatar
	}

	publisher := model.PublisherInfo{Name: name, Avatar: avatar}
	video.Publisher = publisher
	return video, nil
}
