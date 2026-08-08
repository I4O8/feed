package dao

import (
	"feed/model"
)

// ListFeed 查询某用户的 feed，按 id 降序游标分页（cursor 为上一页最后一条 feed.id，0 = 最新一页）
func ListFeed(userID, cursor int64, limit int) ([]*model.Feed, error) {
	list := make([]*model.Feed, 0)
	query := DB.Model(&model.Feed{}).Where("user_id = ?", userID)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if err := query.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// InsertFeedRows 批量插入 feed 记录（同步扩散用）
func InsertFeedRows(rows []model.Feed) error {
	if len(rows) == 0 {
		return nil
	}
	return DB.CreateInBatches(rows, 500).Error
}
