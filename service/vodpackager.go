package service

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"feed/config"
)

// 点播模块：打包器（remux + 分片 + 加密）
// 文档：docs/点播模块.md 4.3 打包流水线 / 4.1 两条并行链路
//
// 工具链（实测确定，见实现记录）：
//   - HLS AES-128 线：纯 ffmpeg（remux + 按段切分片，明文；加密留到分发时按会话现做）
//   - DASH CENC 主线：GPAC MP4Box（ffmpeg 的 -f dash 会「静默忽略」加密参数，
//     只出未加密分片且退出码仍为 0 —— 不可用）
//
// 分发阶段全部是转封装（remux）+ 分片 + 加密，-c copy 语义，码流比特级复制、零质量损失。
// 重编码（含 grain 种子微扰）由 admin 离线产出母版，不进系统流水线。

// HLSResult HLS 一条链（A 或 B）的打包产物
type HLSResult struct {
	Playlist string   // m3u8 相对路径
	Segments []string // 分片相对路径，按段序
}

// Packager 打包器接口：便于替换实现 / 测试
type Packager interface {
	// PackHLS 把母版 remux 成 HLS 明文分片（A/B 各一份），段边界由 starts 指定
	PackHLS(srcPath, outDir string, starts []float64) (*HLSResult, error)
	// PackDASHCENC 把母版 CENC 加密 + 切 DASH 分片，A/B 各一份、共用同一把 key
	PackDASHCENC(srcPath, outDir string, starts []float64, key, kid []byte) (*DASHResult, error)
}

// CommandPackager 基于外部命令的实现（ffmpeg + MP4Box）
type CommandPackager struct{}

// NewCommandPackager 默认打包器
func NewCommandPackager() *CommandPackager { return &CommandPackager{} }

// run 执行外部命令，失败时把 stderr 带回错误
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// 只抓 stderr（ffmpeg/MP4Box 的诊断信息都在 stderr）
	var stderr []byte
	cmd.Stderr = &stderrBuf{&stderr}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s 执行失败: %w\n%s", name, err, string(stderr))
	}
	return nil
}

// stderrBuf 极简 io.Writer：把 stderr 收进 []byte（带上限，避免大文件刷屏）
type stderrBuf struct{ p *[]byte }

func (b *stderrBuf) Write(p []byte) (int, error) {
	const max = 8 << 10
	if len(*b.p) < max {
		*b.p = append(*b.p, p...)
	}
	return len(p), nil
}

// mkdirOut 建输出目录
func mkdirOut(dir string) error { return os.MkdirAll(dir, 0o755) }

// cleanDir 清空并重建输出目录
// 打包必须幂等：重打包时若留着上次的分片，重编号会与残留文件混在一起
// （实测踩过：残留 seg-0..19 + 新写 seg-1..20 → 重编号后变 21 段，与段数不符）
func cleanDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// ── HLS 线：纯 ffmpeg ──

// PackHLS 把 srcPath remux 成 HLS 明文分片，落在 outDir 下
// starts 为段起点（秒）。用 `-f segment`（而非 hls muxer）才能按精确切点分片：
// hls muxer 忽略 -segment_times，只认 -hls_time，会切出与 A/B 不一致的段
// 注意：HLS 线不在这里加密 —— 密钥不烧进媒体文件，加密在每次请求时按会话现做（4.1）
func (p *CommandPackager) PackHLS(srcPath, outDir string, starts []float64) (*HLSResult, error) {
	if err := cleanDir(outDir); err != nil {
		return nil, err
	}
	args := []string{
		"-v", "error", "-y",
		"-i", srcPath,
		"-c", "copy",
		"-f", "segment",
		"-segment_format", "mpegts", // HLS AES-128 走 TS 分片
		"-segment_list", filepath.Join(outDir, "index.m3u8"),
		"-segment_list_type", "m3u8",
	}
	if segTimes := joinFloats(starts); segTimes != "" {
		args = append(args, "-segment_times", segTimes) // 逗号分隔的切点秒表
	}
	args = append(args, filepath.Join(outDir, "seg-%d.ts"))

	if err := run(config.FfmpegBin, args...); err != nil {
		return nil, err
	}
	return collectHLSSegments(outDir)
}

// collectHLSSegments 收集 outDir 下的分片，按段号排序
func collectHLSSegments(outDir string) (*HLSResult, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	segs := make([]string, 0, 64)
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".ts" {
			segs = append(segs, filepath.Join(outDir, name))
		}
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("HLS 打包未产出任何分片")
	}
	sortSegmentPaths(segs)
	return &HLSResult{
		Playlist: filepath.Join(outDir, "index.m3u8"),
		Segments: segs,
	}, nil
}

// ── DASH CENC 线：GPAC MP4Box ──

// DASHResult DASH CENC 一条链的打包产物
type DASHResult struct {
	MPD      string   // MPD 相对路径
	Init     string   // 初始化段（含 tenc/pssh）
	Segments []string // 媒体分片，按段序
}

// PackDASHCENC 用 MP4Box 把 srcPath CENC 加密（cenc-aes-ctr）+ 切 DASH 分片
// A/B 两份必须传同一把 key/kid → 分片互换后仍能解密（文档 4.2 约束 2）
func (p *CommandPackager) PackDASHCENC(srcPath, outDir string, starts []float64, key, kid []byte) (*DASHResult, error) {
	if err := cleanDir(outDir); err != nil {
		return nil, err
	}

	// ① CENC 加密 → 中间文件（单文件 mp4 真加密：含 tenc/senc/pssh）
	encPath := filepath.Join(outDir, "_enc.mp4")
	cryptXML := filepath.Join(outDir, "_drm.xml")
	if err := os.WriteFile(cryptXML, buildCryptXML(key, kid), 0o600); err != nil {
		return nil, err
	}
	if err := run(config.Mp4BoxBin, "-crypt", cryptXML, srcPath, "-out", encPath); err != nil {
		return nil, err
	}

	// ② 切 DASH 分片（-rap 让段边界落在关键帧）
	// ⚠️ 不要加 -profile onDemand：实测它会只产出 init 段、不出媒体分片
	segDur := segmentMillis(starts)
	args := []string{
		"-dash", segDur, "-frag", segDur, "-rap",
		"-segment-name", "seg-$Number$",
		"-out", filepath.Join(outDir, "stream.mpd"),
		encPath,
	}
	if err := run(config.Mp4BoxBin, args...); err != nil {
		return nil, err
	}
	// 中间产物清理（密文已进分片，中间文件无留存价值）
	os.Remove(encPath)
	os.Remove(cryptXML)

	return collectDASHSegments(outDir)
}

// collectDASHSegments 收集 DASH 分片与 init 段，并把段号统一重编号为 0 起
//
// MP4Box 的分片名是 seg-1.m4s … seg-N.m4s（startNumber=1），而 ffmpeg 的 HLS 分片是 seg-0.ts 起。
// 两条链的段号必须对齐（manifest 的段号 = 水印码位序，两端一致，见 4.4「段边界即选片单位」），
// 所以这里把 DASH 分片就地改名 seg-(i).m4s，i 从 0 起。
func collectDASHSegments(outDir string) (*DASHResult, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	res := &DASHResult{MPD: filepath.Join(outDir, "stream.mpd")}
	segs := make([]string, 0, 64)
	for _, e := range entries {
		name := e.Name()
		ext := filepath.Ext(name)
		switch {
		case ext == ".m4s":
			segs = append(segs, filepath.Join(outDir, name))
		case name == "seg-init.mp4" || (len(name) > 4 && name[:4] == "init"):
			res.Init = filepath.Join(outDir, name)
		}
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("DASH 打包未产出任何分片")
	}
	sortSegmentPaths(segs)

	// 重编号：seg-<任意>.m4s → seg-<i>.m4s（i 为排序后的下标，0 起）
	renamed := make([]string, 0, len(segs))
	for i, old := range segs {
		newPath := filepath.Join(outDir, fmt.Sprintf("seg-%d.m4s", i))
		if old != newPath {
			if err := os.Rename(old, newPath); err != nil {
				return nil, fmt.Errorf("DASH 分片重编号失败 %s: %w", old, err)
			}
		}
		renamed = append(renamed, newPath)
	}
	res.Segments = renamed
	return res, nil
}

// buildCryptXML 生成 MP4Box 的 CryptFile
// schema 要点（GPAC 26，实测）：
//   - 每把 key 是 CrypTrack 下的 <key KID="0x.." value="0x.."/> 子节点
//   - 音视频各一条 CrypTrack，否则报 "Missing track crypt info"
//   - KID/value 用 0x 前缀 hex；XML 声明头必须写
func buildCryptXML(key, kid []byte) []byte {
	kidHex := "0x" + hex.EncodeToString(kid)
	keyHex := "0x" + hex.EncodeToString(key)
	zeroIV := "0x00000000000000000000000000000000"
	entry := func(trackID int) string {
		return fmt.Sprintf(`  <CrypTrack trackID="%d" IsEncrypted="1" IV_size="0" first_IV="%s">
    <key KID="%s" value="%s"/>
  </CrypTrack>
`, trackID, zeroIV, kidHex, keyHex)
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<GPACDRM type="CENC AES-CTR">
` + entry(1) + entry(2) + `</GPACDRM>
`)
}

// segmentMillis 段长（毫秒），由段起点推第一个段长；兜底 2000ms
// MP4Box 的 -dash/-frag 只接受单一 duration，段长差异由 -rap 的 IDR 对齐吸收
func segmentMillis(starts []float64) string {
	if len(starts) >= 2 {
		ms := int((starts[1]-starts[0])*1000 + 0.5)
		if ms >= 100 {
			return fmt.Sprintf("%d", ms)
		}
	}
	return "2000"
}

// joinFloats 段起点秒表 → ffmpeg -segment_times 的逗号分隔串（保留 3 位小数）
func joinFloats(fs []float64) string {
	s := ""
	for i, f := range fs {
		if i == 0 {
			continue // 第一段起点是 0，-segment_times 不写 0
		}
		if s != "" {
			s += ","
		}
		s += fmt.Sprintf("%.3f", f)
	}
	return s
}
