package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// requestSpec 场景中的一个接口定义
type requestSpec struct {
	Name         string `json:"name"`         // 统计分组名，如 detail
	Method       string `json:"method"`       // GET/POST/DELETE
	Path         string `json:"path"`         // 支持 {id} 占位符，如 /video/detail、/like/{id}
	Query        string `json:"query"`        // 查询串，支持 {id}/{device}，如 videoId={id}
	Body         string `json:"body"`         // POST 的 JSON body，支持 {id}/{device}
	Auth         bool   `json:"auth"`         // 是否需要携带 JWT
	Weight       int    `json:"weight"`       // 场景内权重
	IDRange      string `json:"idRange"`      // {id} 取值区间，如 "1-200"；空则用全局
	ExpectedCode int    `json:"expectedCode"` // 预期业务码：非 0 时该 code 也算成功（如穿透场景 detail_miss 期望 40400）
}

// scenario 场景定义
type scenario struct {
	Name     string         `json:"name"`
	Requests []requestSpec  `json:"requests"`
}

// loadScenario 从 JSON 文件加载场景
func loadScenario(path string) (*scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取场景文件失败: %w", err)
	}
	var sc scenario
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("解析场景文件失败: %w", err)
	}
	if len(sc.Requests) == 0 {
		return nil, fmt.Errorf("场景 %s 没有定义任何接口", path)
	}
	return &sc, nil
}

// totalWeight 场景权重总和
func (s *scenario) totalWeight() int {
	total := 0
	for _, r := range s.Requests {
		total += r.Weight
	}
	return total
}

// pick 按权重随机选一个接口
func (s *scenario) pick(rng *rand.Rand) *requestSpec {
	total := s.totalWeight()
	if total <= 0 {
		return &s.Requests[rng.Intn(len(s.Requests))]
	}
	n := rng.Intn(total)
	for i := range s.Requests {
		n -= s.Requests[i].Weight
		if n < 0 {
			return &s.Requests[i]
		}
	}
	return &s.Requests[len(s.Requests)-1]
}

// parseRange 解析 "1-200" → [lo, hi]
func parseRange(s string) (int64, int64, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("idRange 格式应为 lo-hi，收到 %q", s)
	}
	lo, err1 := strconv.ParseInt(parts[0], 10, 64)
	hi, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || lo > hi {
		return 0, 0, fmt.Errorf("idRange 解析失败: %q", s)
	}
	return lo, hi, nil
}

// fillPlaceholders 把 {id}/{device} 替换为具体值
func fillPlaceholders(s string, id int64, device string) string {
	s = strings.ReplaceAll(s, "{id}", strconv.FormatInt(id, 10))
	s = strings.ReplaceAll(s, "{device}", device)
	return s
}

// randomDevice 生成随机设备串（reportPlay 去重 key 用，避免压测互相去重）
func randomDevice(rng *rand.Rand) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 16)
	for i := range b {
		b[i] = hex[rng.Intn(len(hex))]
	}
	return string(b)
}

// pickID 从区间随机取一个 id
func pickID(rng *rand.Rand, idRange string, globalRange string) int64 {
	r := idRange
	if r == "" {
		r = globalRange
	}
	if r == "" {
		return 1
	}
	lo, hi, err := parseRange(r)
	if err != nil {
		return 1
	}
	if lo == hi {
		return lo
	}
	return lo + rng.Int63n(hi-lo+1)
}
