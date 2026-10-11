package dao

import (
	"errors"

	"feed/model"

	"gorm.io/gorm"
)

// CreateMovie 新建一条电影记录，返回自增 ID
func CreateMovie(m *model.Movie) (int64, error) {
	if err := DB.Create(m).Error; err != nil {
		return -1, err
	}
	return m.ID, nil
}

// FindMovieByID 按 ID 查电影，查不到返回 (nil, nil)
func FindMovieByID(id int64) (*model.Movie, error) {
	var m *model.Movie
	if err := DB.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

// FindMovieByTmdbID 按外部数据源 ID 查电影（导入去重用）
func FindMovieByTmdbID(tmdbID int64) (*model.Movie, error) {
	var m *model.Movie
	if err := DB.Where("tmdb_id = ?", tmdbID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

// FindMoviesByIDs 批量查电影，避免 N+1（收藏 / 历史列表用）
func FindMoviesByIDs(ids []int64) ([]*model.Movie, error) {
	list := make([]*model.Movie, 0)
	if len(ids) == 0 {
		return list, nil
	}
	if err := DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// MovieListFilter 片库筛选条件
type MovieListFilter struct {
	Sort   string // latest / hot / rating（hot 由 service 层走 Redis，不进这里）
	Genre  string // 类型筛选（JSON 数组里做 LIKE）
	Offset int    // 偏移
	Count  int    // 每页条数
}

// ListMovies 片库列表：只出「就绪」的片子，按 sort 排序，offset 分页
func ListMovies(f MovieListFilter) ([]*model.Movie, error) {
	list := make([]*model.Movie, 0)
	q := DB.Model(&model.Movie{}).Where("status = ?", model.MovieStatusReady)
	if f.Genre != "" {
		// Genres 是 JSON 数组字符串，LIKE 匹配元素（形如 ["动画","剧情"]）
		q = q.Where("genres LIKE ?", "%\""+f.Genre+"\"%")
	}
	switch f.Sort {
	case "rating":
		q = q.Order("rating DESC, id DESC")
	default: // latest
		q = q.Order("id DESC")
	}
	if err := q.Offset(f.Offset).Limit(f.Count).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// SearchMovies 关键词搜索：片名 / 原名 LIKE（P3 简化：不上全文索引）
func SearchMovies(keyword string, offset, count int) ([]*model.Movie, error) {
	list := make([]*model.Movie, 0)
	like := "%" + keyword + "%"
	if err := DB.Model(&model.Movie{}).
		Where("status = ?", model.MovieStatusReady).
		Where("title LIKE ? OR original_title LIKE ?", like, like).
		Order("id DESC").Offset(offset).Limit(count).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// SetMovieSource 上传母版后回写源片路径，并把状态置为「打包中」
func SetMovieSource(id int64, pathA, pathB string) error {
	return DB.Model(&model.Movie{}).Where("id = ?", id).
		Updates(map[string]any{
			"source_path_a": pathA,
			"source_path_b": pathB,
			"status":        model.MovieStatusPacking,
			"fail_reason":   "",
		}).Error
}

// MarkMovieReady 打包完成：回写段信息 + 片长 + 状态就绪
// 片长由打包时 ffprobe 实测写回——NFO 里没有片长字段，不能指望导入时就有
func MarkMovieReady(id int64, durationSec, segmentCount int, durations model.FloatList) error {
	return DB.Model(&model.Movie{}).Where("id = ?", id).
		Updates(map[string]any{
			"duration":          durationSec,
			"segment_count":     segmentCount,
			"segment_durations": durations,
			"status":            model.MovieStatusReady,
			"fail_reason":       "",
		}).Error
}

// MarkMovieFailed 打包失败：记录原因
func MarkMovieFailed(id int64, reason string) error {
	return DB.Model(&model.Movie{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":      model.MovieStatusFailed,
			"fail_reason": reason,
		}).Error
}

// MarkMovieOffline 下架电影
func MarkMovieOffline(id int64) error {
	return DB.Model(&model.Movie{}).Where("id = ?", id).
		Update("status", model.MovieStatusOffline).Error
}

// UpdateMovieMeta 修改元数据（仅允许改元信息字段）
func UpdateMovieMeta(id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return DB.Model(&model.Movie{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteMovie 物理删除电影（与现有表一致，不用软删除）
func DeleteMovie(id int64) error {
	return DB.Delete(&model.Movie{}, id).Error
}

// CountMovies 统计就绪电影总数（分页/统计用，暂留）
func CountMovies() (int64, error) {
	var n int64
	err := DB.Model(&model.Movie{}).Where("status = ?", model.MovieStatusReady).Count(&n).Error
	return n, err
}

// UpdateMoviePlayCount 播放量落库（Redis 脏集合定时任务调用）
func UpdateMoviePlayCount(id int64, play int64) error {
	return DB.Model(&model.Movie{}).Where("id = ?", id).Update("play_count", play).Error
}
