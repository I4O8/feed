package service

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// 点播模块：NFO 导入
// 文档：docs/点播模块.md 3.7 / 十一（原计划走 TMDB 官方 API，本期以本地 NFO 替代，见实现记录）
// NFO 为 Kodi 风 XML：<movie> 根，含 tmdbid / plot / director / actor / genre / rating / title / premiered 等

// ErrNfoInvalid NFO 内容不合法（缺根元素 / 无片名）
var ErrNfoInvalid = errors.New("nfo invalid")

// nfoMovie NFO 文档结构（字段与标签一一对应）
type nfoMovie struct {
	XMLName       xml.Name      `xml:"movie"`
	TmdbID        string        `xml:"tmdbid"`
	ImdbID        string        `xml:"imdbid"`
	Plot          string        `xml:"plot"`
	Outline       string        `xml:"outline"`
	Directors     []nfoDirector `xml:"director"`
	Actors        []nfoActor    `xml:"actor"`
	Genres        []string      `xml:"genre"`
	Rating        string        `xml:"rating"`
	Mpaa          string        `xml:"mpaa"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle"`
	Premiered     string        `xml:"premiered"`
	Year          string        `xml:"year"`
}

// nfoDirector <director tmdbid="...">姓名</director>
type nfoDirector struct {
	TmdbID int64  `xml:"tmdbid,attr"`
	Name   string `xml:",chardata"`
}

// nfoActor <actor><name/><type/><role/><tmdbid/><thumb/><profile/></actor>
type nfoActor struct {
	Name    string `xml:"name"`
	Type    string `xml:"type"`
	Role    string `xml:"role"`
	TmdbID  int64  `xml:"tmdbid"`
	Thumb   string `xml:"thumb"`
	Profile string `xml:"profile"`
}

// parseNfo 解析 NFO 字节流 → 中间结构
func parseNfo(data []byte) (*nfoMovie, error) {
	var doc nfoMovie
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, ErrNfoInvalid
	}
	if strings.TrimSpace(doc.Title) == "" {
		return nil, ErrNfoInvalid
	}
	return &doc, nil
}

// nfoToMovie 中间结构 → model.Movie（含 JSON 字段组装）
func nfoToMovie(doc *nfoMovie) *model.Movie {
	m := &model.Movie{
		Title:         strings.TrimSpace(doc.Title),
		OriginalTitle: strings.TrimSpace(doc.OriginalTitle),
		Description:   strings.TrimSpace(doc.Plot),
		Genres:        model.GenreList(doc.Genres),
		// NFO 无地区字段（mpaa 是分级，不是地区），Region 留空由 admin 手填
	}
	if m.Description == "" {
		m.Description = strings.TrimSpace(doc.Outline)
	}
	m.TmdbID, _ = strconv.ParseInt(strings.TrimSpace(doc.TmdbID), 10, 64)
	m.Rating, _ = strconv.ParseFloat(strings.TrimSpace(doc.Rating), 64)
	m.Year, _ = strconv.Atoi(strings.TrimSpace(doc.Year))
	if m.Year == 0 && len(doc.Premiered) >= 4 { // <year> 缺失时从 <premiered> 前四位取
		m.Year, _ = strconv.Atoi(doc.Premiered[:4])
	}

	// 导演：NFO 中 director 是独立元素，转成 crew（职务 Director）
	for _, d := range doc.Directors {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		m.Crew = append(m.Crew, model.CrewMember{Name: name, TmdbID: d.TmdbID, Job: "Director"})
	}
	// 演员
	for _, a := range doc.Actors {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			continue
		}
		m.Cast = append(m.Cast, model.CastMember{
			Name:    name,
			TmdbID:  a.TmdbID,
			Type:    a.Type,
			Role:    strings.TrimSpace(a.Role),
			Thumb:   strings.TrimSpace(a.Thumb),
			Profile: strings.TrimSpace(a.Profile),
		})
	}
	return m
}

// ImportMovieFromNfo 从 NFO 导入电影元数据
// - nfoData 非空：直接用传入内容
// - nfoData 为空：从 nfoPath 读文件
// 返回电影 ID；tmdbId 已存在则直接返回既有 ID（幂等）
func ImportMovieFromNfo(nfoData []byte, nfoPath string) (int64, error) {
	var raw []byte
	if len(nfoData) > 0 {
		raw = nfoData
	} else {
		b, err := os.ReadFile(nfoPath)
		if err != nil {
			return -1, err
		}
		raw = b
	}

	doc, err := parseNfo(raw)
	if err != nil {
		return -1, err
	}
	m := nfoToMovie(doc)

	// 幂等：同 tmdbId 已入库则直接返回
	if m.TmdbID > 0 {
		if exist, err := dao.FindMovieByTmdbID(m.TmdbID); err != nil {
			return -1, err
		} else if exist != nil {
			return exist.ID, nil
		}
	}
	return dao.CreateMovie(m)
}

// SaveNfoFile 把 NFO 原件落盘留档（data/nfo/<tmdbid>.nfo）
// 返回落盘路径；解析失败或 tmdbId 缺失时返回空串（不阻断导入）
func SaveNfoFile(nfoData []byte, tmdbID int64) (string, error) {
	if tmdbID <= 0 || len(nfoData) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(config.NfoDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(config.NfoDir, strconv.FormatInt(tmdbID, 10)+".nfo")
	if err := os.WriteFile(path, nfoData, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
