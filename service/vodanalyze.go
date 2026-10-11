package service

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"feed/config"
)

// 点播模块：逐片 IDR 分析
// 文档：docs/点播模块.md 4.4
// 为什么必须逐片分析而不是写死段长：每部片子的 GOP 结构不同，
// 写死「固定 N 秒一段」会导致 A/B 段边界对不上，整个水印失效。
// 「一份数据、三处使用」：① 校验 A/B 对齐 ② 导出打包段边界 ③ 导出 manifest 段元信息

// ErrIDRMisalign A/B 两份母版的关键帧序列不逐一相等 → 水印前提不成立
var ErrIDRMisalign = errors.New("A/B 母版 IDR 帧不对齐")

// keyframe 一个关键帧的位置
type keyframe struct {
	Index int     // 第几个关键帧（0 起）
	PTS   float64 // 时间戳（秒）
}

// analyzeKeyframes 用 ffprobe 定位文件里所有 IDR 帧
// 走 packet 级 `flags` 字段（K = keyframe），不解码视频，快
func analyzeKeyframes(path string) ([]keyframe, error) {
	cmd := exec.Command(config.FfprobeBin,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "packet=flags,pts_time",
		"-of", "csv=p=0",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 分析 %s 失败: %w", path, err)
	}

	kfs := make([]keyframe, 0, 64)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// csv 行形如 "0.000000,K__" （不同版本字段顺序可能颠倒，两端都找）
		var ptsStr, flagsStr string
		parts := strings.Split(line, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.Contains(p, "K") {
				flagsStr = p
			} else if p != "" {
				ptsStr = p
			}
		}
		if flagsStr == "" || !strings.Contains(flagsStr, "K") {
			continue
		}
		pts, err := strconv.ParseFloat(ptsStr, 64)
		if err != nil {
			continue
		}
		kfs = append(kfs, keyframe{Index: len(kfs), PTS: pts})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(kfs) == 0 {
		return nil, fmt.Errorf("在 %s 中未找到任何关键帧（非视频流？）", path)
	}
	return kfs, nil
}

// AnalyzeAB 对母版 A、B 各跑一遍 IDR 分析，校验对齐并导出段边界
// 返回每段的起始时间（秒），段数 = len(starts)
// 对齐校验：两份的 IDR 帧号序列必须「逐个相等」——数量相同且每个 PTS 匹配（容差 1ms）
// 不相等 → 返回 ErrIDRMisalign（不能静默放过，对齐不成立则整个水印失效）
//
// 返回的 starts 既用于打包（HLS 的 -segment_times 切点），也用于 manifest 组装，
// 保证 A/B 用同一组切点 → 段边界严格对齐
func AnalyzeAB(pathA, pathB string, targetSegments int) (starts []float64, err error) {
	kfA, err := analyzeKeyframes(pathA)
	if err != nil {
		return nil, fmt.Errorf("母版A: %w", err)
	}
	kfB, err := analyzeKeyframes(pathB)
	if err != nil {
		return nil, fmt.Errorf("母版B: %w", err)
	}

	if len(kfA) != len(kfB) {
		return nil, fmt.Errorf("%w: A 有 %d 个关键帧, B 有 %d 个",
			ErrIDRMisalign, len(kfA), len(kfB))
	}
	for i := range kfA {
		if diff := kfA[i].PTS - kfB[i].PTS; diff > 0.001 || diff < -0.001 {
			return nil, fmt.Errorf("%w: 第 %d 个关键帧 A=%.3fs B=%.3fs",
				ErrIDRMisalign, i+1, kfA[i].PTS, kfB[i].PTS)
		}
	}
	return buildSegmentStarts(kfA, targetSegments), nil
}

// buildSegmentStarts 把关键帧序列切成段：每 perSeg 个关键帧取一个段起点
// 段从某个 IDR 开始 → A/B 换段不会跨帧边界；段数 ≤ 64（水印码是 uint64）
func buildSegmentStarts(kfs []keyframe, targetSegments int) []float64 {
	if targetSegments < 1 {
		targetSegments = 1
	}
	perSeg := len(kfs) / targetSegments
	if perSeg < 1 {
		perSeg = 1
	}
	starts := make([]float64, 0, targetSegments)
	for i := 0; i < len(kfs); i += perSeg {
		starts = append(starts, kfs[i].PTS)
	}
	if len(starts) > 64 {
		starts = starts[:64]
	}
	return starts
}

// SegmentDurations 由段起点推导逐段时长（秒，保留小数）；末段到片尾兜底
// 返回浮点：DASH 的 SegmentTimeline 需要按 timescale 精确表达段长
func SegmentDurations(starts []float64, totalDuration float64) []float64 {
	out := make([]float64, 0, len(starts))
	for i := range starts {
		var dur float64
		if i+1 < len(starts) {
			dur = starts[i+1] - starts[i]
		} else {
			dur = totalDuration - starts[i]
		}
		if dur < 0.001 {
			dur = 0.001
		}
		out = append(out, dur)
	}
	return out
}

// ProbeDuration 用 ffprobe 取文件时长（秒）
func ProbeDuration(path string) (float64, error) {
	cmd := exec.Command(config.FfprobeBin, "-v", "error",
		"-show_entries", "format=duration", "-of", "csv=p=0", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}
	return d, nil
}
