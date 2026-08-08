package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var (
		url          = flag.String("url", "http://localhost:8080", "服务基地址")
		path         = flag.String("path", "", "单接口模式：路径，支持 {id}，如 /video/detail")
		method       = flag.String("method", "GET", "单接口模式：GET/POST/DELETE")
		query        = flag.String("query", "", "单接口模式：查询串，如 videoId={id}")
		body         = flag.String("body", "", "单接口模式：POST JSON body，支持 {id}/{device}")
		concurrency  = flag.Int("concurrency", 100, "并发 worker 数（支持 1000+）")
		duration     = flag.Duration("duration", 60*time.Second, "压测时长，如 60s/300s")
		warmup       = flag.Duration("warmup", 5*time.Second, "预热时长（不计统计）")
		rate         = flag.Float64("rate", 0, "全局限速 req/s，0=不限")
		timeout      = flag.Duration("timeout", 10*time.Second, "单请求超时")
		token        = flag.String("token", "", "固定 JWT（带鉴权接口用）")
		tokenFile    = flag.String("token-file", "", "token 池文件（每行 userID token）")
		scenarioOpt  = flag.String("scenario", "", "场景名（加载 ../scenarios/<name>.json）；置空=单接口模式")
		scenarioFile = flag.String("scenario-file", "", "场景文件路径（默认 ../scenarios/<scenario>.json）")
		idRange      = flag.String("id-range", "1-50000", "{id} 全局取值区间")
		expectedCode = flag.Int("expected-code", 0, "单接口模式预期业务码：非 0 时该 code 也算成功（如 detail_miss 用 40400）")
		version      = flag.String("version", "opt", "输出文件名标签：opt/base")
		outputDir    = flag.String("output-dir", "../results", "JSON 输出目录")
		keepAlive    = flag.Bool("keepalive", true, "HTTP 连接复用")
		notes        = flag.String("notes", "", "附加环境说明（写进 JSON）")
	)
	flag.Parse()

	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "concurrency 必须为正数")
		os.Exit(1)
	}

	// 场景 or 单接口
	var sc *scenario
	var scenarioName string
	if *scenarioOpt != "" {
		sp := *scenarioFile
		if sp == "" {
			sp = filepath.Join("..", "scenarios", *scenarioOpt+".json")
		}
		var err error
		sc, err = loadScenario(sp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "场景加载失败: %v\n", err)
			os.Exit(1)
		}
		scenarioName = sc.Name
		fmt.Printf("场景 %s：%d 个接口，权重和 %d\n", scenarioName, len(sc.Requests), sc.totalWeight())
	} else {
		if *path == "" {
			fmt.Fprintln(os.Stderr, "单接口模式需要 -path；或使用 -scenario 指定场景")
			os.Exit(1)
		}
		scenarioName = "single"
	}

	// token 池
	var tokens []string
	if *token != "" {
		tokens = []string{*token}
	} else if *tokenFile != "" {
		t, err := readTokens(*tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "token 文件读取失败: %v\n", err)
			os.Exit(1)
		}
		tokens = t
	}

	cfg := &config{
		BaseURL:     *url,
		Method:      *method,
		Path:        *path,
		Query:       *query,
		Body:        *body,
		Concurrency: *concurrency,
		Duration:    *duration,
		Warmup:      *warmup,
		Rate:        *rate,
		Timeout:     *timeout,
		KeepAlive:   *keepAlive,
		Token:        *token,
		TokenPool:    tokens,
		IDRange:      *idRange,
		ExpectedCode: *expectedCode,
		Scenario:     sc,
	}

	client := buildHTTPClient(*timeout, *concurrency, *keepAlive)
	rl := newRateLimiter(*rate)
	stats := newStatsRegistry()
	workers := make([]*worker, *concurrency)
	for i := range workers {
		workers[i] = newWorker(cfg, client, rl, stats, int64(i)*7919+time.Now().UnixNano()%100000)
	}

	// 预热（把 detail 缓存喂热，不计统计）
	if *warmup > 0 {
		fmt.Printf("预热 %s（不计统计）...\n", *warmup)
		runFor(workers, *warmup)
		fmt.Println("预热完成")
	}

	// 正式压测
	for _, w := range workers {
		w.enabled.Store(true)
	}
	start := time.Now()
	done := make(chan struct{})
	go progressReporter(stats, done)
	runFor(workers, *duration)
	close(done)
	elapsed := time.Since(start)

	// 汇总
	per := make(map[string]summary)
	for _, name := range stats.order {
		per[name] = stats.get(name).summarize(elapsed)
	}
	res := resultFile{
		Version:     *version,
		Scenario:    scenarioName,
		URL:         *url,
		Start:       start.Format(time.RFC3339),
		End:         time.Now().Format(time.RFC3339),
		DurationS:   elapsed.Seconds(),
		Concurrency: *concurrency,
		Rate:        *rate,
		WarmupS:     (*warmup).Seconds(),
		Total:       stats.all.summarize(elapsed),
		PerEndpoint: per,
		Notes:       *notes,
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}
	outPath := filepath.Join(*outputDir, fmt.Sprintf("result-%s-%s.json", *version, scenarioName))
	if err := writeResult(outPath, res); err != nil {
		fmt.Fprintf(os.Stderr, "写结果失败: %v\n", err)
		os.Exit(1)
	}

	// 终端汇总
	s := res.Total
	fmt.Printf("\n===== 压测完成: %s / %s =====\n", *version, scenarioName)
	fmt.Printf("总请求 %d · QPS %.1f · 平均 %.2fms · P50 %dms · P95 %dms · P99 %dms\n",
		s.Total, s.QPS, s.AvgMS, s.P50MS, s.P95MS, s.P99MS)
	fmt.Printf("错误 %d (%.2f%%) · 超时 %d\n", s.Errors, s.ErrorRatePct, s.Timeouts)
	fmt.Printf("结果已写入: %s\n", outPath)
}

// progressReporter 每 5s 打印瞬时 QPS
func progressReporter(stats *statsRegistry, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	lastTotal := stats.totalNow()
	lastTime := time.Now()
	for {
		select {
		case <-done:
			return
		case now := <-ticker.C:
			total := stats.totalNow()
			elapsed := now.Sub(lastTime).Seconds()
			instant := float64(total-lastTotal) / elapsed
			lastTotal = total
			lastTime = now
			fmt.Printf("\r[进行中] 总请求 %d · 瞬时 QPS %.0f        ", total, instant)
		}
	}
}

// readTokens 读取 token 文件（每行 `userID token`，取最后一列）
func readTokens(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tokens []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		tokens = append(tokens, fields[len(fields)-1])
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("token 文件为空: %s", path)
	}
	return tokens, nil
}
