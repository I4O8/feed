package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// config 压测全局配置（CLI 参数解析后的运行配置）
type config struct {
	BaseURL     string        // 服务基地址，如 http://localhost:8080
	Method      string        // 单接口模式：方法
	Path        string        // 单接口模式：路径（支持 {id}）
	Query       string        // 单接口模式：查询串
	Body        string        // 单接口模式：POST body
	Concurrency int           // 并发 worker 数
	Duration    time.Duration // 压测时长
	Warmup      time.Duration // 预热时长（不计统计）
	Rate        float64       // 全局限速 req/s，0=不限
	Timeout     time.Duration // 单请求超时
	KeepAlive   bool          // 连接复用
	Token       string        // 固定 token
	TokenPool   []string      // token 池（seed 产出，轮询取）
	IDRange     string        // {id} 全局取值区间
	ExpectedCode int          // 单接口模式预期业务码（非 0 时该码算成功）
	Scenario    *scenario     // 场景（nil = 单接口模式）
}

// rateLimiter 全局 token bucket 限速
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func newRateLimiter(rate float64) *rateLimiter {
	if rate <= 0 {
		return &rateLimiter{}
	}
	return &rateLimiter{interval: time.Duration(float64(time.Second) / rate)}
}

func (rl *rateLimiter) allow() {
	if rl.interval <= 0 {
		return
	}
	for {
		rl.mu.Lock()
		now := time.Now()
		if rl.next.IsZero() {
			rl.next = now
		}
		wait := rl.next.Sub(now)
		if wait <= 0 {
			rl.next = now.Add(rl.interval)
			rl.mu.Unlock()
			return
		}
		rl.mu.Unlock()
		time.Sleep(wait)
	}
}

// statsRegistry 汇总所有接口统计（含整体）
type statsRegistry struct {
	mu    sync.Mutex
	all   *endpointStats // 整体统计
	items map[string]*endpointStats
	order []string
}

func newStatsRegistry() *statsRegistry {
	return &statsRegistry{
		all:   newEndpointStats(),
		items: make(map[string]*endpointStats),
	}
}

// get 取或建一个接口分组的统计
func (r *statsRegistry) get(name string) *endpointStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.items[name]; ok {
		return s
	}
	s := newEndpointStats()
	r.items[name] = s
	r.order = append(r.order, name)
	return s
}

// worker 单个并发协程
type worker struct {
	cfg     *config
	client  *http.Client
	rl      *rateLimiter
	rng     *rand.Rand
	tokens  atomic.Uint64 // token 池轮询游标
	stats   *statsRegistry
	enabled atomic.Bool // false = warmup 阶段，不记录统计
}

// buildHTTPClient 构造调优后的 HTTP 客户端（1000+ 并发前提）
func buildHTTPClient(timeout time.Duration, concurrency int, keepAlive bool) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        concurrency,
		MaxIdleConnsPerHost: concurrency,
		MaxConnsPerHost:     concurrency,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   !keepAlive,
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

// newWorker 创建 worker
func newWorker(cfg *config, client *http.Client, rl *rateLimiter, stats *statsRegistry, seed int64) *worker {
	w := &worker{
		cfg:    cfg,
		client: client,
		rl:     rl,
		rng:    rand.New(rand.NewSource(seed)),
		stats:  stats,
	}
	w.enabled.Store(false)
	return w
}

// record 记录一次请求（warmup 阶段丢弃）
func (w *worker) record(name string, dur time.Duration, err errorType) {
	if !w.enabled.Load() {
		return
	}
	w.stats.all.record(dur, err)
	if name != "" {
		w.stats.get(name).record(dur, err)
	}
}

// run 持续发请求直到 stopAt
func (w *worker) run(stopAt time.Time) {
	for {
		if time.Now().After(stopAt) {
			return
		}
		w.rl.allow()
		w.doOne()
	}
}

// doOne 执行一次请求并记录
func (w *worker) doOne() {
	var spec *requestSpec
	name := ""
	if w.cfg.Scenario != nil {
		spec = w.cfg.Scenario.pick(w.rng)
		name = spec.Name
	} else {
		spec = &requestSpec{
			Method:       w.cfg.Method,
			Path:         w.cfg.Path,
			Query:        w.cfg.Query,
			Body:         w.cfg.Body,
			Auth:         w.cfg.Token != "" || len(w.cfg.TokenPool) > 0,
			ExpectedCode: w.cfg.ExpectedCode,
		}
	}

	// 占位符替换
	id := pickID(w.rng, spec.IDRange, w.cfg.IDRange)
	device := randomDevice(w.rng)
	path := fillPlaceholders(spec.Path, id, device)
	query := fillPlaceholders(spec.Query, id, device)
	body := fillPlaceholders(spec.Body, id, device)

	url := w.cfg.BaseURL + path
	if query != "" {
		url += "?" + query
	}

	// 请求构造
	var reader io.Reader
	if body != "" && (spec.Method == "POST" || spec.Method == "PUT" || spec.Method == "PATCH") {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(spec.Method, url, reader)
	if err != nil {
		w.record(name, 0, errNetwork)
		return
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if spec.Auth {
		req.Header.Set("Authorization", "Bearer "+w.pickToken())
	}

	// 执行
	start := time.Now()
	resp, err := w.client.Do(req)
	dur := time.Since(start)
	if err != nil {
		w.record(name, dur, classifyError(err))
		return
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		w.record(name, dur, errNetwork)
		return
	}

	// 判定成功：本项目错误一律 HTTP 200 + body code
	if resp.StatusCode != http.StatusOK {
		w.record(name, dur, errHTTPStatus)
		return
	}
	var parsed struct {
		Code int `json:"code"`
	}
	if json.Unmarshal(respBody, &parsed) != nil {
		w.record(name, dur, errNetwork)
		return
	}
	// 默认 code==0 算成功；spec 指定了预期业务码（如穿透场景的 40400）时，该码也算成功
	expect := spec.ExpectedCode
	if expect != 0 {
		if parsed.Code != expect {
			w.record(name, dur, errAppCode)
			return
		}
	} else if parsed.Code != 0 {
		w.record(name, dur, errAppCode)
		return
	}
	w.record(name, dur, errNone)
}

// pickToken 从 token 池轮询取一个（按 worker 各自游标，分散 userId）
func (w *worker) pickToken() string {
	if w.cfg.Token != "" {
		return w.cfg.Token
	}
	if len(w.cfg.TokenPool) == 0 {
		return ""
	}
	idx := w.tokens.Add(1) % uint64(len(w.cfg.TokenPool))
	return w.cfg.TokenPool[idx]
}

// classifyError 把 Do 返回的错误分类为超时/网络
func classifyError(err error) errorType {
	if errors.Is(err, context.DeadlineExceeded) {
		return errTimeout
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errTimeout
	}
	return errNetwork
}

// runFor 让所有 worker 跑 d 时长（阻塞等待收尾）
func runFor(workers []*worker, d time.Duration) {
	stopAt := time.Now().Add(d)
	var wg sync.WaitGroup
	for _, w := range workers {
		wg.Add(1)
		go func(w *worker) {
			defer wg.Done()
			w.run(stopAt)
		}(w)
	}
	wg.Wait()
}
