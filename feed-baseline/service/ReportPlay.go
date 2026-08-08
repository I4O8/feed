package service

import (
	"feed/dao"
)

// ReportPlay 播放上报（退化版：无设备去重，每次直接 MySQL 原子自增）
func ReportPlay(videoID int64, deviceID string) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}
	return dao.ReportPlay(videoID)
}
