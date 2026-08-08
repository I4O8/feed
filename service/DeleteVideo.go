package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"feed/dao"
)

var ErrPermissionDenied = errors.New("permission denied")

// DeleteVideo 删除视频：校验归属 → 删磁盘文件 → 删数据库记录 → 清缓存
func DeleteVideo(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}
	if video.Uploader == userID {
		vDst := filepath.Join("./data", strings.TrimPrefix(video.VideoURL, "/uploads/"))
		cDst := filepath.Join("./data", strings.TrimPrefix(video.CoverURL, "/uploads/"))
		if err := os.Remove(vDst); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		if err := os.Remove(cDst); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		if err := dao.DeleteVideo(videoID); err != nil {
			return err
		}
		delCache(videoID) // 清旧快照，防缓存里残留幽灵视频
		return nil
	}
	return ErrPermissionDenied
}
