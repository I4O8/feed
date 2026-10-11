package service

import (
	"context"
	"strconv"

	"feed/dao"
	"feed/model"
)

// 点播模块：片库浏览 / 详情 / 搜索 / 播放上报
// 文档：docs/点播模块.md 8.2 浏览

const movieCountPrefix = "movie:count:" // 电影计数 hash：field play

// fillMoviePlay 从 Redis 计数填播放数（Redis 挂了静默保留 MySQL 兜底）
func fillMoviePlay(m *model.Movie) {
	v, err := dao.RDB.HGet(context.Background(),
		movieCountPrefix+strconv.FormatInt(m.ID, 10), "play").Result()
	if err != nil {
		return
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		m.PlayCount = n
	}
}

// VodList 片库列表：最新 / 热门 / 评分 + 分类筛选 + 分页
// 热门榜复用现有 Redis 计数模型（与 video 同一套路，不重造）
func VodList(sortBy, genre string, offset, count int) ([]*model.Movie, int, error) {
	if sortBy == "hot" {
		return vodListHot(genre, offset, count)
	}
	list, err := dao.ListMovies(dao.MovieListFilter{Sort: sortBy, Genre: genre, Offset: offset, Count: count})
	if err != nil {
		return nil, -1, err
	}
	for _, m := range list {
		fillMoviePlay(m)
	}
	next := 0
	if len(list) == count {
		next = offset + count
	}
	return list, next, nil
}

// vodListHot 热门榜：按 Redis 播放数排序（简化实现：取一批就绪片子后按播放数排）
func vodListHot(genre string, offset, count int) ([]*model.Movie, int, error) {
	// 取足够宽的一批（热门榜不需要全量精确，播放数来自 Redis 计数）
	wide := offset + count
	if wide < 200 {
		wide = 200
	}
	all, err := dao.ListMovies(dao.MovieListFilter{Sort: "latest", Genre: genre, Offset: 0, Count: wide})
	if err != nil {
		return nil, -1, err
	}
	for _, m := range all {
		fillMoviePlay(m)
	}
	// 稳定排序：播放数降序，同数按 id 降序
	sortMoviesByPlay(all)
	if offset >= len(all) {
		return []*model.Movie{}, 0, nil
	}
	end := offset + count
	if end > len(all) {
		end = len(all)
	}
	page := all[offset:end]
	next := 0
	if end < len(all) {
		next = end
	}
	return page, next, nil
}

// VodDetail 电影详情：完整元数据 + 我是否收藏 + 我的观看进度（登录时）
func VodDetail(movieID, uid int64) (*model.Movie, error) {
	m, err := dao.FindMovieByID(movieID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrVodMovieNotFound
	}
	fillMoviePlay(m)
	if uid > 0 {
		if favored, _ := dao.MovieFavoriteExists(uid, movieID); favored {
			m.Favored = true
		}
		if p, _ := dao.FindMovieProgress(uid, movieID); p != nil {
			m.Position = p.Position
		}
	}
	return m, nil
}

// VodSearch 关键词搜索
func VodSearch(keyword string, offset, count int) ([]*model.Movie, int, error) {
	list, err := dao.SearchMovies(keyword, offset, count)
	if err != nil {
		return nil, -1, err
	}
	for _, m := range list {
		fillMoviePlay(m)
	}
	next := 0
	if len(list) == count {
		next = offset + count
	}
	return list, next, nil
}

// VodReportPlay 播放上报：Redis 计数 +1，打脏标记由定时任务落库
// 与 video 的「Redis 权威 + 定时落库」同一套路
func VodReportPlay(movieID int64) error {
	m, err := dao.FindMovieByID(movieID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrVodMovieNotFound
	}
	key := movieCountPrefix + strconv.FormatInt(movieID, 10)
	dao.RDB.HIncrBy(context.Background(), key, "play", 1)
	dao.RDB.SAdd(context.Background(), "movie:dirty", movieID)
	return nil
}

// VodFav 收藏 / 取消收藏
func VodFav(uid, movieID int64) error {
	if m, _ := dao.FindMovieByID(movieID); m == nil {
		return ErrVodMovieNotFound
	}
	return dao.InsertMovieFavorite(&model.MovieFavorite{UserID: uid, MovieID: movieID})
}

func VodUnFav(uid, movieID int64) error {
	return dao.DeleteMovieFavorite(uid, movieID)
}

func VodFavStatus(uid, movieID int64) (bool, error) {
	return dao.MovieFavoriteExists(uid, movieID)
}

// VodFavList 我的电影收藏：按收藏时间倒序游标分页（cursor = 上一页最后一条 favorites.id）
func VodFavList(uid, cursor int64, count int) ([]*model.Movie, int64, error) {
	favs, err := dao.ListMovieFavorites(uid, cursor, count)
	if err != nil {
		return nil, -1, err
	}
	ids := make([]int64, 0, len(favs))
	for _, f := range favs {
		ids = append(ids, f.MovieID)
	}
	movies, err := dao.FindMoviesByIDs(ids)
	if err != nil {
		return nil, -1, err
	}
	byID := make(map[int64]*model.Movie, len(movies))
	for _, m := range movies {
		byID[m.ID] = m
	}
	// 按收藏顺序重排（FindMoviesByIDs 不保证顺序）
	out := make([]*model.Movie, 0, len(favs))
	for _, f := range favs {
		if m := byID[f.MovieID]; m != nil {
			fillMoviePlay(m)
			out = append(out, m)
		}
	}
	var next int64
	if len(favs) == count {
		next = favs[len(favs)-1].ID
	}
	return out, next, nil
}

// VodReportProgress 上报进度（upsert）
func VodReportProgress(uid, movieID int64, position int) error {
	m, _ := dao.FindMovieByID(movieID)
	if m == nil {
		return ErrVodMovieNotFound
	}
	return dao.UpsertMovieProgress(&model.MovieProgress{
		UserID:   uid,
		MovieID:  movieID,
		Position: position,
		Duration: m.Duration,
	})
}

// VodProgress 取单片进度（续播定位）
func VodProgress(uid, movieID int64) (int, error) {
	p, err := dao.FindMovieProgress(uid, movieID)
	if err != nil || p == nil {
		return 0, err
	}
	return p.Position, nil
}

// VodHistoryItem 观看历史条目（进度 + 电影元数据）
type VodHistoryItem struct {
	Movie    *model.Movie `json:"movie"`
	Position int          `json:"position"`
	Progress float64      `json:"progress"` // 百分比 0~100
}

// VodHistory 观看历史：按最近观看倒序
func VodHistory(uid, cursor int64, count int) ([]*VodHistoryItem, int64, error) {
	rows, err := dao.ListMovieProgress(uid, cursor, count)
	if err != nil {
		return nil, -1, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.MovieID)
	}
	movies, err := dao.FindMoviesByIDs(ids)
	if err != nil {
		return nil, -1, err
	}
	byID := make(map[int64]*model.Movie, len(movies))
	for _, m := range movies {
		byID[m.ID] = m
	}
	items := make([]*VodHistoryItem, 0, len(rows))
	for _, r := range rows {
		m := byID[r.MovieID]
		if m == nil {
			continue
		}
		fillMoviePlay(m)
		prog := 0.0
		if r.Duration > 0 {
			prog = float64(r.Position) / float64(r.Duration) * 100
		}
		items = append(items, &VodHistoryItem{Movie: m, Position: r.Position, Progress: prog})
	}
	var next int64
	if len(rows) == count {
		next = rows[len(rows)-1].UpdatedAt.Unix()
	}
	return items, next, nil
}

// sortMoviesByPlay 播放数降序、同数按 id 降序（稳定可复现）
func sortMoviesByPlay(list []*model.Movie) {
	// 用插入排序避免引 sort 包做闭包 —— 列表不大，且要稳定
	for i := 1; i < len(list); i++ {
		cur := list[i]
		j := i - 1
		for j >= 0 && lessMovie(cur, list[j]) {
			list[j+1] = list[j]
			j--
		}
		list[j+1] = cur
	}
}

func lessMovie(a, b *model.Movie) bool {
	if a.PlayCount != b.PlayCount {
		return a.PlayCount > b.PlayCount
	}
	return a.ID > b.ID
}
