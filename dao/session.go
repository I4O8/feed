package dao

import (
	"errors"
	"time"

	"feed/model"

	"gorm.io/gorm"
)

// CreateSession 新建播放会话，返回自增 ID
func CreateSession(s *model.PlaybackSession) (int64, error) {
	if err := DB.Create(s).Error; err != nil {
		return -1, err
	}
	return s.ID, nil
}

// FindSessionByID 按 ID 取会话；查不到返回 (nil, nil)
func FindSessionByID(id int64) (*model.PlaybackSession, error) {
	var s *model.PlaybackSession
	if err := DB.First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// TouchSessionHeartbeat 心跳：刷新心跳时间
func TouchSessionHeartbeat(id int64, at time.Time) error {
	return DB.Model(&model.PlaybackSession{}).Where("id = ?", id).
		Update("heartbeat_at", at).Error
}

// SetSessionWatermark 回写会话水印码（水印码在会话落库后才生成，需二次更新）
func SetSessionWatermark(id int64, code uint64) error {
	return DB.Model(&model.PlaybackSession{}).Where("id = ?", id).
		Update("watermark_code", code).Error
}

// EndSession 结束会话：置状态为结束 + 记录结束时间
// status 由调用方指定（0=正常结束 2=被吊销）
func EndSession(id int64, status int8) error {
	now := time.Now()
	return DB.Model(&model.PlaybackSession{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "ended_at": now}).Error
}

// RevokeSessionsByUID 吊销某用户全部活跃会话（发现盗号 / 越权时立刻踢人）
func RevokeSessionsByUID(uid int64) error {
	now := time.Now()
	return DB.Model(&model.PlaybackSession{}).
		Where("uid = ? AND status = ?", uid, model.SessionActive).
		Updates(map[string]any{"status": model.SessionRevoked, "ended_at": now}).Error
}

// CountActiveSessions 统计用户活跃会话数（兜底校准，正常并发判定走 Redis）
func CountActiveSessions(uid int64) (int64, error) {
	var n int64
	err := DB.Model(&model.PlaybackSession{}).
		Where("uid = ? AND status = ?", uid, model.SessionActive).Count(&n).Error
	return n, err
}

// FindSessionByWatermark 溯源：按水印码反查会话（拿到泄露片还原出码后定位到人）
func FindSessionByWatermark(code uint64) ([]*model.PlaybackSession, error) {
	list := make([]*model.PlaybackSession, 0)
	if err := DB.Where("watermark_code = ?", code).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
