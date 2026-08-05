package service

import (
	"feed/dao"
)

// ReportPlay 播放上报：先确认视频存在，再播放量 +1；不存在返回 ErrVideoNotFound
func ReportPlay(videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}
	return dao.ReportPlay(videoID)
}
