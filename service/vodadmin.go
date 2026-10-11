package service

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// 点播模块：管理端（上架 / 上传母版 / 查状态）
// 文档：docs/点播模块.md 8.1 / 6.4 管理端鉴权

// ErrAdminKeyInvalid admin 密钥无效
var ErrAdminKeyInvalid = errors.New("admin key invalid")

// CheckAdminKey 常量时间比较 admin 密钥（防时序侧信道 —— 信安细节）
func CheckAdminKey(got string) bool {
	want := config.VodAdminKey
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// VodImportNfo 按 NFO 导入电影元数据（替代 TMDB 刮削，见实现记录）
// nfoData 非空则用它；否则从 nfoPath 读
func VodImportNfo(nfoData []byte, nfoPath string) (int64, error) {
	id, err := ImportMovieFromNfo(nfoData, nfoPath)
	if err != nil {
		return -1, err
	}
	// NFO 原件留档
	m, _ := dao.FindMovieByID(id)
	if m != nil {
		SaveNfoFile(nfoData, m.TmdbID)
	}
	return id, nil
}

// VodCreateMovie 手动新建电影元数据
func VodCreateMovie(m *model.Movie) (int64, error) {
	if strings.TrimSpace(m.Title) == "" {
		return -1, fmt.Errorf("片名不能为空")
	}
	m.Status = model.MovieStatusPacking // 待上传源片
	return dao.CreateMovie(m)
}

// VodUpdateMovie 修改元数据（仅允许元信息字段）
func VodUpdateMovie(id int64, fields map[string]any) error {
	m, _ := dao.FindMovieByID(id)
	if m == nil {
		return ErrVodMovieNotFound
	}
	// 白名单：只放行业务字段，防止改 status/路径
	allow := map[string]bool{
		"title": true, "original_title": true, "description": true,
		"poster_url": true, "backdrop_url": true, "duration": true,
		"year": true, "region": true, "rating": true,
	}
	safe := map[string]any{}
	for k, v := range fields {
		if allow[k] {
			safe[k] = v
		}
	}
	return dao.UpdateMovieMeta(id, safe)
}

// VodDeleteMovie 下架 / 删除
func VodDeleteMovie(id int64) error {
	m, _ := dao.FindMovieByID(id)
	if m == nil {
		return ErrVodMovieNotFound
	}
	return dao.DeleteMovie(id)
}

// VodMovieStatus 查打包状态（前端轮询用）
func VodMovieStatus(id int64) (status int8, reason string, segmentCount int, err error) {
	m, err := dao.FindMovieByID(id)
	if err != nil {
		return 0, "", 0, err
	}
	if m == nil {
		return 0, "", 0, ErrVodMovieNotFound
	}
	return m.Status, m.FailReason, m.SegmentCount, nil
}

// VodUploadSource 上传母版 A + 母版 B → 落盘 + 建/更新 movie 源路径 → 触发打包
// 文档 8.1：两份成对、缺一不可；系统不做重编码
func VodUploadSource(movieID int64, fileA, fileB *multipart.FileHeader) error {
	m, err := dao.FindMovieByID(movieID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrVodMovieNotFound
	}
	if fileA == nil || fileB == nil {
		return fmt.Errorf("母版 A 与 B 必须成对上传")
	}

	srcDir := filepath.Join(config.VodDir, "src", fmt.Sprint(movieID))
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}
	pathA := filepath.Join(srcDir, "master_a"+extOf(fileA.Filename))
	pathB := filepath.Join(srcDir, "master_b"+extOf(fileB.Filename))
	if err := saveUpload(fileA, pathA); err != nil {
		return err
	}
	if err := saveUpload(fileB, pathB); err != nil {
		return err
	}

	// 回写源路径 + 状态置打包中
	if err := dao.SetMovieSource(movieID, pathA, pathB); err != nil {
		return err
	}
	// 丢 MQ 任务，立即返回（前端轮询 status）
	return PublishPackTask(movieID)
}

// extOf 取扩展名（默认 .mp4）
func extOf(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ".mp4"
	}
	return ext
}

// saveUpload 把上传文件写到指定路径
func saveUpload(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}
