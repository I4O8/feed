package main

import (
	"encoding/json"
	"os"
)

// resultFile 压测结果文件结构
type resultFile struct {
	Version     string             `json:"version"`     // opt / base
	Scenario    string             `json:"scenario"`    // mix / detail_hot / ...
	URL         string             `json:"url"`         // 被压服务地址
	Start       string             `json:"start"`       // 开始时间（RFC3339）
	End         string             `json:"end"`         // 结束时间
	DurationS   float64            `json:"duration_s"`  // 实际压测时长（秒）
	Concurrency int                `json:"concurrency"`
	Rate        float64            `json:"rate"`
	WarmupS     float64            `json:"warmup_s"`
	Total       summary            `json:"total"`        // 整体指标
	PerEndpoint map[string]summary `json:"per_endpoint"` // 场景模式各接口细分
	Notes       string             `json:"notes,omitempty"`
}

// writeResult 序列化结果到 JSON 文件
func writeResult(path string, r resultFile) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
