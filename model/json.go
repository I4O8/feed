package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// jsonScan JSON 列统一读入口：兼容 MySQL 驱动回传的 []byte / string
func jsonScan(dst any, v any) error {
	if v == nil {
		return nil
	}
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		return errors.New("不支持的 JSON 列类型")
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dst)
}

// jsonValue JSON 列统一写出口：结构体 → JSON 字符串
func jsonValue(src any) (driver.Value, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// ── 以下均为 GORM JSON 列类型：实现 Scanner / Valuer，让复杂字段直接落 JSON ──

// GenreList 类型列表（movie.genres）
type GenreList []string

func (g *GenreList) Scan(v any) error          { return jsonScan(g, v) }
func (g GenreList) Value() (driver.Value, error) { return jsonValue(g) }

// IntList 整数数组（预留：整数型 JSON 数组列）
type IntList []int

func (l *IntList) Scan(v any) error            { return jsonScan(l, v) }
func (l IntList) Value() (driver.Value, error) { return jsonValue(l) }

// FloatList 浮点数组（movie.segment_durations，逐段时长秒，需小数精度）
// 用浮点而非整数：DASH 的 SegmentTimeline 要按 timescale 精确表达段长，
// 整数秒取整会让 timeline 与实际分片错位
type FloatList []float64

func (l *FloatList) Scan(v any) error            { return jsonScan(l, v) }
func (l FloatList) Value() (driver.Value, error) { return jsonValue(l) }

// CastMember 演员（含角色），存于 movie.cast
// 字段名对齐 Kodi NFO 的 <actor> 块，便于 NFO 导入直接落库
type CastMember struct {
	Name    string `json:"name"`    // 演员名
	TmdbID  int64  `json:"tmdbid"`  // 外部数据源人物 ID
	Type    string `json:"type"`    // Actor / Director 等
	Role    string `json:"role"`    // 饰演角色
	Thumb   string `json:"thumb"`   // 人物头像
	Profile string `json:"profile"` // 人物主页
}

// CastList 演员列表（movie.cast）
type CastList []CastMember

func (c *CastList) Scan(v any) error            { return jsonScan(c, v) }
func (c CastList) Value() (driver.Value, error) { return jsonValue(c) }

// CrewMember 职员（含职务），存于 movie.crew
type CrewMember struct {
	Name   string `json:"name"`   // 姓名
	TmdbID int64  `json:"tmdbid"` // 外部数据源人物 ID
	Job    string `json:"job"`    // 职务：Director / Writer 等
}

// CrewList 职员列表（movie.crew）
type CrewList []CrewMember

func (c *CrewList) Scan(v any) error            { return jsonScan(c, v) }
func (c CrewList) Value() (driver.Value, error) { return jsonValue(c) }
