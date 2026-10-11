package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// 点播模块：打包流水线编排
// 文档：docs/点播模块.md 4.3
//
// 流程：
//   ① 逐片分析 A/B 的 IDR 帧号 → 校验对齐 + 导出段边界
//   ② remux + 分片（A、B 各一遍，段边界由 ① 导出）
//   ③ 加密打包（CENC / AES-128；A/B 用同一把 content key）
//   ④ 回写 movie_rendition + content_key + movie.segment_count
//   ⑤ movie 状态 = 就绪
//
// AI 实现说明：① 的分析结果同时喂给 ② 的切点与后续 manifest 组装（一份数据三处用）。

// PackMovie 打包一部电影（由 MQ worker 调用；也可直接调用做测试）
func PackMovie(movieID int64) error {
	m, err := dao.FindMovieByID(movieID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("影片 %d 不存在", movieID)
	}
	if m.SourcePathA == "" || m.SourcePathB == "" {
		dao.MarkMovieFailed(movieID, "缺少母版 A/B")
		return fmt.Errorf("影片 %d 缺少母版 A/B", movieID)
	}

	// ① 逐片 IDR 分析 + A/B 对齐校验
	starts, err := AnalyzeAB(m.SourcePathA, m.SourcePathB, config.VodTargetSegments)
	if err != nil {
		dao.MarkMovieFailed(movieID, err.Error())
		return err
	}
	dur, _ := ProbeDuration(m.SourcePathA)
	durations := SegmentDurations(starts, dur) // []float64

	// ③ 先生成内容密钥（A/B 共用），信封加密落库
	key, kid, err := GenContentKey()
	if err != nil {
		dao.MarkMovieFailed(movieID, "密钥生成失败: "+err.Error())
		return err
	}
	sealed, nonce, err := SealContentKey(key)
	if err != nil {
		dao.MarkMovieFailed(movieID, "密钥封装失败: "+err.Error())
		return err
	}
	keyID, err := dao.CreateContentKey(&model.ContentKey{
		KeyID:    KidHex(kid),
		KeyValue: sealed,
		Nonce:    nonce,
		Status:   1,
	})
	if err != nil {
		dao.MarkMovieFailed(movieID, "密钥入库失败: "+err.Error())
		return err
	}

	// ② + ③ 打包两条链（A/B 各一份）
	root := filepath.Join(config.VodDir, fmt.Sprint(movieID))
	pk := NewCommandPackager()

	// 重打包前清掉旧产物行（幂等重跑）
	if err := dao.DeleteRenditionsByMovie(movieID); err != nil {
		dao.MarkMovieFailed(movieID, "清理旧产物失败: "+err.Error())
		return err
	}

	// ── HLS 线（ffmpeg，明文分片；加密在分发时按会话现做）──
	hlsA, err := pk.PackHLS(m.SourcePathA, filepath.Join(root, "hls", "a"), starts)
	if err != nil {
		dao.MarkMovieFailed(movieID, "HLS-A 打包失败: "+err.Error())
		return err
	}
	hlsB, err := pk.PackHLS(m.SourcePathB, filepath.Join(root, "hls", "b"), starts)
	if err != nil {
		dao.MarkMovieFailed(movieID, "HLS-B 打包失败: "+err.Error())
		return err
	}
	// 硬校验：分片数必须 == 段数，否则水印码位序与分片错位（A/B 互换也会失败）
	if len(hlsA.Segments) != len(starts) || len(hlsB.Segments) != len(starts) {
		msg := fmt.Sprintf("HLS 分片数与段数不一致: A=%d B=%d 段=%d",
			len(hlsA.Segments), len(hlsB.Segments), len(starts))
		dao.MarkMovieFailed(movieID, msg)
		return errors.New(msg)
	}
	if err := saveRendition(movieID, model.ContainerHLS, model.DRMSchemeAES128,
		hlsA.Playlist, hlsA.Segments, hlsB.Segments, keyID); err != nil {
		dao.MarkMovieFailed(movieID, "HLS 产物入库失败: "+err.Error())
		return err
	}

	// ── DASH CENC 线（MP4Box，打包时一次性加密）──
	dashA, err := pk.PackDASHCENC(m.SourcePathA, filepath.Join(root, "dash", "a"), starts, key, kid)
	if err != nil {
		dao.MarkMovieFailed(movieID, "DASH-A 打包失败: "+err.Error())
		return err
	}
	dashB, err := pk.PackDASHCENC(m.SourcePathB, filepath.Join(root, "dash", "b"), starts, key, kid)
	if err != nil {
		dao.MarkMovieFailed(movieID, "DASH-B 打包失败: "+err.Error())
		return err
	}
	// 硬校验：A/B 分片数必须相等且 == 段数（水印可互换的前提，也是文档 4.2 的工程约束）
	if len(dashA.Segments) != len(starts) || len(dashB.Segments) != len(starts) {
		msg := fmt.Sprintf("DASH 分片数与段数不一致: A=%d B=%d 段=%d",
			len(dashA.Segments), len(dashB.Segments), len(starts))
		dao.MarkMovieFailed(movieID, msg)
		return errors.New(msg)
	}
	if err := saveRendition(movieID, model.ContainerDASH, model.DRMSchemeClearKey,
		dashA.MPD, dashA.Segments, dashB.Segments, keyID); err != nil {
		dao.MarkMovieFailed(movieID, "DASH 产物入库失败: "+err.Error())
		return err
	}

	// ④ + ⑤ 回写片长 / 段信息 + 状态就绪
	if err := dao.MarkMovieReady(movieID, int(dur+0.5), len(starts), model.FloatList(durations)); err != nil {
		return err
	}
	return nil
}

// saveRendition 写一条 rendition 行
// 用「分片集合的首个分片目录」推出路径模式（含 %d 段号占位），供 manifest 选片用
func saveRendition(movieID int64, container, drmScheme, playlist string,
	segsA, segsB []string, contentKeyID int64) error {

	patA, extA := segmentPattern(segsA)
	patB, _ := segmentPattern(segsB)
	if patA == "" || patB == "" {
		return fmt.Errorf("分片集合为空")
	}
	r := &model.MovieRendition{
		MovieID:         movieID,
		Resolution:      "source", // 母版直出，不降码率（见文档「不做分发侧重编码」）
		Container:       container,
		DRMScheme:       drmScheme,
		PlaylistPath:    playlist,
		SegmentPatternA: patA,
		SegmentPatternB: patB,
		ContentKeyID:    contentKeyID,
	}
	if extA != "" {
		r.Codec = extToCodec(extA)
	}
	_, err := dao.CreateRendition(r)
	return err
}

// segmentPattern 从分片路径集合推出「按段号填充」的路径模式
// seg-1.ts → 目录/seg-%d.ts
func segmentPattern(segs []string) (pattern, ext string) {
	if len(segs) == 0 {
		return "", ""
	}
	first := segs[0]
	dir := filepath.Dir(first)
	base := filepath.Base(first)
	ext = filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	// 去掉结尾数字，留下列名前缀
	prefix := stem
	for len(prefix) > 0 && prefix[len(prefix)-1] >= '0' && prefix[len(prefix)-1] <= '9' {
		prefix = prefix[:len(prefix)-1]
	}
	return filepath.Join(dir, prefix+"%d"+ext), ext
}

// extToCodec 分片扩展名 → 粗略 codec 标注
func extToCodec(ext string) string {
	switch ext {
	case ".ts", ".m4s":
		return "h264"
	default:
		return ""
	}
}

// VodMovieDir 影片产物根目录
func VodMovieDir(movieID int64) string {
	return filepath.Join(config.VodDir, fmt.Sprint(movieID))
}

// EnsureVodDirs 首次启动时建好点播目录（分片 / 海报 / NFO / 母版）
func EnsureVodDirs() error {
	for _, d := range []string{
		config.VodDir,
		filepath.Join(config.VodDir, "src"),
		config.PosterDir,
		config.NfoDir,
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
