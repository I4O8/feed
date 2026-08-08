package main

import (
	"sync/atomic"
	"time"
)

// 延迟直方图：1ms 一桶，0..10000ms（10s）；超过 10s 进 overflow
// 用分桶而非存全量样本：300s × 数万 QPS 的样本量级也能压进 ~80KB 内存
const (
	histogramBuckets = 10001
	histogramMaxMS   = 10000
)

// errorType 错误分类
type errorType int

const (
	errNone       errorType = iota // 成功（code==0）
	errHTTPStatus                  // HTTP 状态码非 200
	errAppCode                     // HTTP 200 但业务 code != 0
	errNetwork                     // 连接/IO 错误
	errTimeout                     // 单请求超时
)

// endpointStats 单个接口的统计（原子计数，worker 并发写）
type endpointStats struct {
	total     atomic.Int64 // 总请求数
	success   atomic.Int64 // 成功数（code==0）
	totalLat  atomic.Int64 // 延迟累加（ns）
	histogram [histogramBuckets]atomic.Uint64
	overflow  atomic.Uint64 // 延迟 > 10s 的请求数
	errors    atomic.Int64
	timeouts  atomic.Int64
}

func newEndpointStats() *endpointStats {
	return &endpointStats{}
}

func (s *endpointStats) record(dur time.Duration, err errorType) {
	s.total.Add(1)
	if err == errNone {
		s.success.Add(1)
	} else {
		s.errors.Add(1)
		if err == errTimeout {
			s.timeouts.Add(1)
		}
	}
	s.totalLat.Add(int64(dur))
	ms := dur.Milliseconds()
	if ms > histogramMaxMS {
		s.overflow.Add(1)
	} else {
		s.histogram[ms].Add(1)
	}
}

// summary 单个接口的汇总
type summary struct {
	Total        int64   `json:"total_requests"`
	Success      int64   `json:"success"`
	QPS          float64 `json:"qps"`
	AvgMS        float64 `json:"avg_ms"`
	P50MS        int64   `json:"p50_ms"`
	P95MS        int64   `json:"p95_ms"`
	P99MS        int64   `json:"p99_ms"`
	Errors       int64   `json:"errors"`
	ErrorRatePct float64 `json:"error_rate_pct"`
	Timeouts     int64   `json:"timeouts"`
}

// summarize 生成汇总（elapsed 为实际压测时长）
func (s *endpointStats) summarize(elapsed time.Duration) summary {
	total := s.total.Load()
	sum := summary{
		Total:    total,
		Success:  s.success.Load(),
		Errors:   s.errors.Load(),
		Timeouts: s.timeouts.Load(),
	}
	if elapsed > 0 {
		sum.QPS = float64(total) / elapsed.Seconds()
	}
	if total > 0 {
		sum.AvgMS = float64(s.totalLat.Load()) / float64(total) / 1e6
		sum.ErrorRatePct = float64(s.errors.Load()) / float64(total) * 100
	}
	sum.P50MS = s.percentile(50)
	sum.P95MS = s.percentile(95)
	sum.P99MS = s.percentile(99)
	return sum
}

// totalNow 当前整体总请求数（进度打印用）
func (r *statsRegistry) totalNow() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.all.total.Load()
}

// percentile 从累计分布反查百分位延迟（毫秒）
func (s *endpointStats) percentile(p int) int64 {
	total := s.total.Load()
	if total == 0 {
		return 0
	}
	rank := int64(float64(total) * float64(p) / 100)
	var cum uint64
	for i := 0; i < histogramBuckets; i++ {
		cum += s.histogram[i].Load()
		if int64(cum) >= rank {
			return int64(i)
		}
	}
	return histogramMaxMS + 1 // 落在 overflow 桶
}
