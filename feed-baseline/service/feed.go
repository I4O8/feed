package service

import (
	"feed/dao"
	"feed/model"
)

// Feed 基于 MySQL feed 表的关注流（退化版：无 Redis List）
// offset 语义 = feed 游标（上一页最后一条 feed.id，0 = 最新一页）
// 组装退化：每条 feed 逐条查视频 + 逐条查发布者（N+1），计数读 video 列
func Feed(userID, offset int64, count int) ([]*model.Video, int64, error) {
	rows, err := dao.ListFeed(userID, offset, count)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []*model.Video{}, 0, nil
	}

	videos := make([]*model.Video, 0, len(rows))
	for _, r := range rows {
		v, err := dao.FindByVideoID(r.VideoID)
		if err != nil {
			return nil, 0, err
		}
		if v == nil {
			continue // 视频被物理删除：自动跳过
		}
		account, err := dao.FindByID(v.Uploader)
		if err != nil {
			return nil, 0, err
		}
		if account == nil {
			v.Publisher = model.PublisherInfo{Name: "已注销用户", Avatar: ""}
		} else {
			v.Publisher = model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
		}
		videos = append(videos, v)
	}

	var nextCursor int64
	if len(rows) == count {
		nextCursor = rows[len(rows)-1].ID
	}
	return videos, nextCursor, nil
}

// fanout 视频入库后把 video_id 推给上传者的所有粉丝
// 退化版：同步查粉丝 + 批量 INSERT feed 表（优化版走 MQ 异步 + Redis pipeline）
func fanout(videoID, uploaderID int64) error {
	fans, err := dao.ListFollowers(uploaderID)
	if err != nil {
		return err
	}
	if len(fans) == 0 {
		return nil
	}
	rows := make([]model.Feed, 0, len(fans))
	for _, fanID := range fans {
		rows = append(rows, model.Feed{UserID: fanID, VideoID: videoID})
	}
	return dao.InsertFeedRows(rows)
}
