package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// 点播模块：per-user manifest 生成（水印的「载体轴」）
// 文档：docs/点播模块.md 4.2 载体轴 / 6.5 变体访问控制
//
// 核心：每个会话的 playlist 内容都不同 —— 按会话水印码逐段选 A 还是 B，
// 且每个分片 URL 带独立签名（签名覆盖变体位，改体位即失效）。
// 所以 manifest 不再是静态文件，而是「带鉴权的动态接口」。

// SegRef 一个分片的引用（已按水印码选定变体 + 已签名）
type SegRef struct {
	No     int     // 段号（0 起）
	URL    string  // 带签名的分片 URL
	Variant byte   // 'a' / 'b'（本会话在该段拿到的变体）
	Dur    float64 // 段时长（秒，精确值用于 MPD 的 SegmentTimeline）
	Start  float64 // 段起始时刻（秒，绝对，用于 SegmentTimeline 的 t）
}

// manifestSegs 按水印码逐段选变体 + 生成签名 URL
// 分片 URL 同时带 ticket 与签名：ticket 证明会话真实存在（防伪造 sessionId），
// 签名证明「变体位 + 段号」未被篡改（见 6.5）
//
// starts 是打包时逐片分析导出的段起点（秒），与母版的 IDR 边界严格对应 ——
// DASH 的 SegmentTimeline 与 HLS 的 #EXTINF 都用它，保证 manifest 与实际分片一致
func manifestSegs(ticket string, movieID, renditionID, sessionID int64, wmCode uint64, starts []float64, totalDur float64) []SegRef {
	exp := nowUnix() + int64(config.VodSegmentSigTTL)
	out := make([]SegRef, 0, len(starts))
	for no := range starts {
		variant := WatermarkVariant(wmCode, no)
		sig := SignSegment(movieID, renditionID, sessionID, no, variant, exp)
		// 分片 URL 带上 ticket：播放器直接吃 manifest 里的 URL，不会额外补参数，
		// 而 /vod/segment 要用 ticket 做「会话真实存在」的强绑定（防伪造 sessionId）
		url := fmt.Sprintf("%s/vod/segment?ticket=%s&m=%d&r=%d&s=%d&n=%d&v=%c&exp=%d&sig=%s",
			config.PublicBaseURL, ticket, movieID, renditionID, sessionID, no, variant, exp, sig)
		// 段长：到下一段起点；末段到片尾
		var dur float64
		if no+1 < len(starts) {
			dur = starts[no+1] - starts[no]
		} else {
			dur = totalDur - starts[no]
		}
		if dur < 0.001 {
			dur = 0.001
		}
		out = append(out, SegRef{No: no, URL: url, Variant: variant, Dur: dur, Start: starts[no]})
	}
	return out
}

// BuildHLSPlaylist 生成 per-user m3u8
// - #EXT-X-KEY 的 key URI 是 per-session 的（ticket 鉴权）
// - 每个分片 URL 带签名（覆盖变体位）
// - 加密在服务端按会话现做（明文分片在磁盘，见 4.1 HLS 线模型）
func BuildHLSPlaylist(ticket string, segs []SegRef) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	maxDur := 0.0
	for _, s := range segs {
		if s.Dur > maxDur {
			maxDur = s.Dur
		}
	}
	// TARGETDURATION 必须 ≥ 实际最长段（HLS 规范），向上取整
	target := int(maxDur + 0.999)
	if target < 1 {
		target = 6
	}
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", target)
	// per-session 密钥：本会话唯一的一把 key（绝对 URL，hls 子播放器同样按绝对地址取）
	fmt.Fprintf(&b, "#EXT-X-KEY:METHOD=AES-128,URI=\"%s/vod/key?ticket=%s\"\n", config.PublicBaseURL, ticket)
	for _, s := range segs {
		fmt.Fprintf(&b, "#EXTINF:%.3f,\n%s\n", s.Dur, s.URL)
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return b.String()
}

// BuildDASHMPD 生成 per-user MPD（SegmentList + ClearKey ContentProtection）
// 用 SegmentList 而非 SegmentTemplate：因为每段的 URL 逐用户不同，模板表达不了
//
// ⚠️ 三个「shaka 能播」的硬要求（都对 shaka 源码核实过）：
//  1. Laurl 命名空间必须是 DASH-IF 规范值 http://dashif.org/guidelines/clearKey
//     （shaka 的 ClearKeyNamespaceUri_ 按「命名空间 URI」匹配，别的前缀找不到 → 拿不到 license → 黑屏）；
//     ContentProtection 的 schemeIdUri 必须是 urn:uuid:e2719d58-a985-b3c9-781a-b030af78d30e。
//  2. 多段 SegmentList 必须给出段时长：或 SegmentList@duration，或 SegmentTimeline。
//     否则 shaka 抛 CRITICAL DASH_NO_SEGMENT_INFO（见 lib/dash/segment_list.js 的 checkSegmentListInfo_）。
//  3. 每段时间必须一致：shaka 用 timeline[i].end 当第 i 段的结束（与 mediaSegments 按下标配对，
//     timeline 长度必须 == SegmentURL 数量）。所以这里逐段写 <S t= d=>，不合并 r。
//  4. Period 必须有 duration：否则 timeShiftBufferDepth 算成 NaN。
func BuildDASHMPD(ticket string, segs []SegRef, kidHex string, totalDur float64) string {
	const (
		timescale = 1000 // 用毫秒精度表达秒级段长，避免整数秒取整误差
	)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013" `+
		`xmlns:clearkey="http://dashif.org/guidelines/clearKey" `+
		`profiles="urn:mpeg:dash:profile:isoff-on-demand:2011" type="static" `+
		`mediaPresentationDuration="PT%.3fS" minBufferTime="PT2S">`+"\n", totalDur)
	fmt.Fprintf(&b, ` <Period duration="PT%.3fS">`+"\n", totalDur)
	b.WriteString(`  <AdaptationSet mimeType="video/mp4" startWithSAP="1" segmentAlignment="true">` + "\n")

	// CENC 声明（default_KID 用 dash 的 uuid 形式：8-4-4-4-12）
	fmt.Fprintf(&b, `   <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" cenc:default_KID="%s"/>`+"\n", dashKID(kidHex))
	// ClearKey：license 端点（带 ticket）
	b.WriteString(`   <ContentProtection schemeIdUri="urn:uuid:e2719d58-a985-b3c9-781a-b030af78d30e" value="ClearKey1.0">` + "\n")
	fmt.Fprintf(&b, `    <clearkey:Laurl Lic_type="EME-1.0">%s</clearkey:Laurl>`+"\n",
		xmlEsc(config.PublicBaseURL+"/vod/license?ticket="+ticket))
	b.WriteString("   </ContentProtection>\n")

	fmt.Fprintf(&b, `   <Representation id="1" bandwidth="1000000" codecs="avc1.64001f,mp4a.40.2">`+"\n")
	fmt.Fprintf(&b, `    <SegmentList timescale="%d" duration="%d">`+"\n", timescale, durScaled(segs[0].Dur, timescale))
	// init 段也带 ticket + 签名（同一套校验逻辑），变体位用 'a' 占位（init 与变体无关）
	initExp := nowUnix() + int64(config.VodSegmentSigTTL)
	initSig := SignSegment(-1, -1, -1, -1, 'a', initExp)
	fmt.Fprintf(&b, `     <Initialization sourceURL="%s"/>`+"\n",
		xmlEsc(fmt.Sprintf("%s/vod/segment?ticket=%s&kind=init&exp=%d&sig=%s",
			config.PublicBaseURL, ticket, initExp, initSig)))
	// SegmentTimeline：逐段一个 <S>，t = 段起点、d = 段长（都按 timescale 缩放）
	// timeline 长度必须 == SegmentURL 数量（shaka 按下标配对）
	b.WriteString("     <SegmentTimeline>\n")
	for _, s := range segs {
		fmt.Fprintf(&b, `      <S t="%d" d="%d"/>`+"\n", durScaled(s.Start, timescale), durScaled(s.Dur, timescale))
	}
	b.WriteString("     </SegmentTimeline>\n")
	for _, s := range segs {
		fmt.Fprintf(&b, `     <SegmentURL media="%s"/>`+"\n", xmlEsc(s.URL))
	}
	b.WriteString("    </SegmentList>\n")
	b.WriteString("   </Representation>\n")
	b.WriteString("  </AdaptationSet>\n")
	b.WriteString(" </Period>\n")
	b.WriteString("</MPD>\n")
	return b.String()
}

// durScaled 秒 → 按 timescale 缩放的整数（四舍五入）
func durScaled(sec float64, timescale int) int {
	return int(sec*float64(timescale) + 0.5)
}

// dashKID 16 字节 hex → dash 的 8-4-4-4-12 uuid 形式
func dashKID(hexStr string) string {
	if len(hexStr) != 32 {
		return hexStr
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:32])
}

// xmlEsc XML 属性值转义（URL 里的 & 必须转义，否则 MPD 解析失败）
func xmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

// ── 分片文件定位 ──

// segmentFile 由 rendition 的路径模式 + 段号 + 变体，定位磁盘上的分片文件
// SegmentPatternA/B 形如 .../seg-%d.ts（%d 为段号占位）
func segmentFile(rend *model.MovieRendition, segNo int, variant byte) string {
	pat := rend.SegmentPatternA
	if variant == 'b' {
		pat = rend.SegmentPatternB
	}
	return fmt.Sprintf(pat, segNo)
}

// initFile DASH 初始化段路径（与 A 变体同目录的 init 段）
func initFile(rend *model.MovieRendition) string {
	dir := filepath.Dir(fmt.Sprintf(rend.SegmentPatternA, 1))
	// MP4Box 产出 seg-init.mp4，与其分片同目录
	for _, name := range []string{"seg-init.mp4", "init.mp4"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadSegmentStarts 取影片逐段「起点」序列（秒），供 manifest 组装用
// 由逐段时长累加还原 —— 打包时 starts 已由 IDR 边界导出、end 时刻即下段起点，
// 故累加可无损还原（含末段）。缺失段信息时按片长均分兜底。
func LoadSegmentStarts(movie *model.Movie) []float64 {
	if len(movie.SegmentDurations) > 0 {
		out := make([]float64, 0, len(movie.SegmentDurations))
		var acc float64
		for _, d := range movie.SegmentDurations {
			out = append(out, acc)
			acc += d
		}
		return out
	}
	n := movie.SegmentCount
	if n < 1 {
		n = 1
	}
	each := float64(movie.Duration) / float64(n)
	if each < 0.001 {
		each = 1
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = float64(i) * each
	}
	return out
}

// RecordVariantAbuse 记录「越权采集」异常信号（文档 6.5 第 4 条）
// 同会话请求了非本会话分配的变体位 → 记 Redis 计数，可触发告警 / 限速
func RecordVariantAbuse(sessionID int64) {
	key := "vod:abuse:" + fmt.Sprint(sessionID)
	dao.RDB.Incr(context.Background(), key)
	dao.RDB.Expire(context.Background(), key, 24*time.Hour)
}
