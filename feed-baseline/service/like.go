package service

import (
	"feed/dao"
	"feed/model"
)

// Like 点赞（退化版：直接写 MySQL likes 表 + 同步更新 video.like_count，无 Redis 状态键）
func Like(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	// 幂等：已赞直接返回（并发下可能都判未赞 → 都 INSERT，唯一键吞其一）
	exists, err := dao.LikeExists(userID, videoID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := dao.InsertLike(&model.Likes{UserID: userID, VideoID: videoID}); err != nil {
		return err
	}
	return dao.IncrLikeCount(videoID)
}

// UnLike 取消点赞
func UnLike(userID, videoID int64) error {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return err
	}
	if video == nil {
		return ErrVideoNotFound
	}

	deleted, err := dao.DeleteLike(&model.Likes{UserID: userID, VideoID: videoID})
	if err != nil {
		return err
	}
	if !deleted {
		return nil // 本来就没赞，幂等
	}
	return dao.DecrLikeCount(videoID)
}

// LikeStatus 是否已点赞：直接查 MySQL likes 表
func LikeStatus(userID, videoID int64) (bool, error) {
	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return false, err
	}
	if video == nil {
		return false, ErrVideoNotFound
	}
	return dao.LikeExists(userID, videoID)
}
