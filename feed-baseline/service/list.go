package service

import (
	"feed/dao"
	"feed/model"
)

// List 视频列表：游标分页，可选按发布者过滤（uploaderID 为 0 = 全站）
// 返回已组装 Publisher 的视频切片、下一页游标（0 = 没有更多）
// 退化版：发布者逐条查询（N+1），计数直接读 video 列
func List(uploaderID, cursor int64, limit int) ([]*model.Video, int64, error) {
	videos, err := dao.ListVideos(uploaderID, cursor, limit)
	if err != nil {
		return nil, -1, err
	}

	// 逐条查发布者（N+1：每条视频查一次 user 表）
	for i := range videos {
		account, err := dao.FindByID(videos[i].Uploader)
		if err != nil {
			return nil, -1, err
		}
		if account == nil {
			videos[i].Publisher = model.PublisherInfo{Name: "已注销用户", Avatar: ""}
		} else {
			videos[i].Publisher = model.PublisherInfo{Name: account.Name, Avatar: account.Avatar}
		}
	}

	// 取满 limit 才说明还有更旧的，nextCursor = 最后一条 id；没取满说明到底，nextCursor = 0
	var nextCursor int64
	if len(videos) == limit {
		nextCursor = videos[len(videos)-1].ID
	}
	return videos, nextCursor, nil
}
