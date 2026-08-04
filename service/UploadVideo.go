package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"feed/config"
	"feed/dao"
	"feed/model"
)

// ErrDamagedVideo 视频文件损坏，无法解析出有效时长
var (
	ErrDamagedVideo       = errors.New("Video Info Damaged")
	ErrCoverCaptureFailed = errors.New("Cover Capture Failed")
)

// mediaInfo 用 ffprobe 读取视频时长，四舍五入为秒；解析不出时长视为视频损坏
func mediaInfo(dst string) (int, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		dst,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}

	// 解析 ffprobe 输出的 JSON，取 format.duration（字符串表示的秒数）
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return 0, err
	}
	secFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}

	totalSec := int(math.Round(secFloat))
	if totalSec == 0 {
		return 0, ErrDamagedVideo
	}
	return totalSec, nil
}

// captureCover 未上传封面时自动截图作为封面
func captureCover(dst string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "tempCover*.jpg")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	cmd := exec.Command(
		"ffmpeg",
		"-i", dst,
		"-ss", "00:00:01",
		"-vframes", "1",
		"-f", "image2",
		"-y",
		tmpPath,
	)
	if err := cmd.Run(); err != nil {
		return nil, ErrCoverCaptureFailed
	}
	cover, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}
	return cover, nil
}

// UploadVideo 上传视频：视频流式落盘并 ffprobe 取时长，封面落盘（未传时由调用方决定），最后写入 video 表
// 参数：video 视频字节流、videoName 视频文件名、cover 封面数据（nil 表示未传）、coverName 封面文件名、
// userID 发布者ID、title 标题、describe 描述
// 返回：视频ID、视频URL、封面URL、时长（秒）
func UploadVideo(video io.Reader, videoName string, cover []byte, coverName string, userID int64, title, describe string) (int64, string, string, int, error) {
	// 视频处理
	vExt := filepath.Ext(videoName)
	newVideoName := generateRandomString(30) + vExt
	videoURL := "/uploads/video/" + newVideoName

	// 确保目录存在，流式落盘（边读边写，内存占用恒定）
	if err := os.MkdirAll(config.VideoDir, 0o755); err != nil {
		return -1, "", "", -1, err
	}
	videoSavePath := filepath.Join(config.VideoDir, newVideoName)
	f, err := os.Create(videoSavePath)
	if err != nil {
		return -1, "", "", -1, err
	}
	defer f.Close()
	if _, err := io.Copy(f, video); err != nil {
		return -1, "", "", -1, err
	}

	// ffprobe 提取时长
	duration, err := mediaInfo(videoSavePath)
	if err != nil {
		return -1, "", "", -1, err
	}

	// 封面处理
	cExt := filepath.Ext(coverName)
	if cover == nil {
		cover, err = captureCover(videoSavePath)
		cExt = ".jpg"
		if err != nil {
			return -1, "", "", -1, err
		}
	}

	newCoverName := generateRandomString(20) + cExt
	coverURL := "/uploads/video/cover/" + newCoverName

	if err := os.MkdirAll(config.CoverDir, 0o755); err != nil {
		return -1, "", "", -1, err
	}
	coverSavePath := filepath.Join(config.CoverDir, newCoverName)
	if err := os.WriteFile(coverSavePath, cover, 0o644); err != nil {
		return -1, "", "", -1, err
	}

	// 写库
	tmp := model.Video{Title: title, CoverURL: coverURL, VideoURL: videoURL, Describe: describe, Uploader: userID, Duration: duration}
	videoID, err := dao.CreateVideo(&tmp)
	if err != nil {
		return -1, "", "", -1, err
	}
	return videoID, videoURL, coverURL, duration, nil
}
