package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 造一份测试母版：GOP 2s，20s 长 → 10 个 IDR。
// seed 不同 → 画面颗粒不同，模拟「grain 种子微扰」的 A/B 两份母版。
func makeMaster(t *testing.T, path string, seed int) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440",
		"-t", "20",
		"-c:v", "libx264", "-g", "50", "-keyint_min", "50", "-sc_threshold", "0",
		"-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest",
		"-metadata", "comment=seed"+string(rune('0'+seed)),
		path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("造母版失败: %v\n%s", err, out)
	}
}

// TestAnalyzeABAligned 校验 A/B 对齐分析：同参数两份应逐帧对齐，段数 > 1
func TestAnalyzeABAligned(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")
	makeMaster(t, a, 1)
	makeMaster(t, b, 2)

	starts, err := AnalyzeAB(a, b, 8)
	if err != nil {
		t.Fatalf("AnalyzeAB 失败: %v", err)
	}
	if len(starts) < 2 {
		t.Fatalf("段数太少: %d", len(starts))
	}
	if starts[0] != 0 {
		t.Fatalf("首段起点应为 0，实际 %v", starts[0])
	}
	t.Logf("段数=%d 起点=%v", len(starts), starts)

	dur, err := ProbeDuration(a)
	if err != nil {
		t.Fatalf("ProbeDuration 失败: %v", err)
	}
	t.Logf("时长=%.2fs 逐段时长=%v", dur, SegmentDurations(starts, dur))
}

// TestAnalyzeABMismatch 故意用不同 GOP 的两份，必须报对齐失败
func TestAnalyzeABMismatch(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")
	makeMaster(t, a, 1)
	// B 用不同 GOP（1s）→ IDR 数翻倍 → 应当不齐
	cmd := exec.Command("ffmpeg", "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=25", "-f", "lavfi", "-i", "sine=frequency=440",
		"-t", "20", "-c:v", "libx264", "-g", "25", "-keyint_min", "25", "-sc_threshold", "0",
		"-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", b)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("造 B 失败: %v\n%s", err, out)
	}
	if _, err := AnalyzeAB(a, b, 8); err == nil {
		t.Fatal("GOP 不同的两份母版应当报对齐失败，但没有")
	} else {
		t.Logf("按预期报错: %v", err)
	}
}

// TestPackBothChains 端到端跑两条打包链，校验产物形态
func TestPackBothChains(t *testing.T) {
	if _, err := exec.LookPath("MP4Box"); err != nil {
		t.Skip("未安装 MP4Box，跳过 CENC 链")
	}
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	b := filepath.Join(dir, "b.mp4")
	makeMaster(t, a, 1)
	makeMaster(t, b, 2)

	starts, err := AnalyzeAB(a, b, 8)
	if err != nil {
		t.Fatalf("AnalyzeAB 失败: %v", err)
	}
	pk := NewCommandPackager()

	// HLS 线
	hlsA, err := pk.PackHLS(a, filepath.Join(dir, "hls", "a"), starts)
	if err != nil {
		t.Fatalf("PackHLS 失败: %v", err)
	}
	if len(hlsA.Segments) != len(starts) {
		t.Errorf("HLS 分片数 %d 与段数 %d 不一致", len(hlsA.Segments), len(starts))
	}
	t.Logf("HLS 分片数=%d", len(hlsA.Segments))

	// DASH CENC 线（A/B 同一把 key）
	key, kid, _ := GenContentKey()
	dashA, err := pk.PackDASHCENC(a, filepath.Join(dir, "dash", "a"), starts, key, kid)
	if err != nil {
		t.Fatalf("PackDASHCENC 失败: %v", err)
	}
	dashB, err := pk.PackDASHCENC(b, filepath.Join(dir, "dash", "b"), starts, key, kid)
	if err != nil {
		t.Fatalf("PackDASHCENC(B) 失败: %v", err)
	}
	if len(dashA.Segments) == 0 {
		t.Fatal("DASH 未产出分片")
	}
	// 校验 init 段真的带 tenc，且 KID 与传入一致
	assertTencKID(t, dashA.Init, kid)
	t.Logf("DASH 分片数=%d init=%s", len(dashA.Segments), filepath.Base(dashA.Init))

	// 关键：A/B 两份分片数必须一致（互换的前提）
	if len(dashA.Segments) != len(dashB.Segments) {
		t.Errorf("A/B DASH 分片数不一致: %d vs %d", len(dashA.Segments), len(dashB.Segments))
	}
	// 每片必须自包含 senc（IV 自包含 → 可互换）
	assertSenc(t, dashA.Segments[0])
}

// assertTencKID 按 box 结构解析 tenc，校验 IsProtected=1 且 KID 一致
//
// tenc 是 FullBox，tag 之后的字节布局（ISO 23001-7）：
//
//	[0:4]   version(1) + flags(3)
//	[4]     reserved
//	[5]     reserved（version 0）或 crypt/skip byte block（version 1）
//	[6]     default_isProtected
//	[7]     default_Per_Sample_IV_Size
//	[8:24]  default_KID（16 字节）
func assertTencKID(t *testing.T, path string, kid []byte) {
	t.Helper()
	d, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 init 段失败: %v", err)
	}
	i := indexOf(d, []byte("tenc"))
	if i < 0 {
		t.Fatal("init 段缺少 tenc box（未加密？）")
	}
	body := d[i+4:] // tenc 的 body 从 tag 之后开始
	if len(body) < 24 {
		t.Fatal("tenc box 过短")
	}
	if body[6] != 1 {
		t.Errorf("IsProtected 应为 1，实际 %d", body[6])
	}
	if body[7] != 16 {
		t.Errorf("IV_size 应为 16，实际 %d", body[7])
	}
	gotKID := body[8:24]
	for j := range kid {
		if gotKID[j] != kid[j] {
			t.Fatalf("KID 不匹配: got %x want %x", gotKID, kid)
		}
	}
}

// assertSenc 校验分片含 senc box（每样本 IV 自包含）
func assertSenc(t *testing.T, path string) {
	t.Helper()
	d, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读分片失败: %v", err)
	}
	if indexOf(d, []byte("senc")) < 0 {
		t.Error("分片缺少 senc box（IV 未自包含，A/B 互换会解不开）")
	}
}

// indexOf 子串查找（bytes.Index 的简写，避免多引一个包）
func indexOf(haystack, needle []byte) int {
	n := len(needle)
	for i := 0; i+n <= len(haystack); i++ {
		match := true
		for j := 0; j < n; j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
