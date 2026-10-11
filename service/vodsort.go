package service

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 点播模块：分片路径排序
// 分片名形如 seg-1.ts / seg-2.ts ... seg-10.ts，字典序会把 10 排在 2 前，
// 所以要按「名里的数字」数值排序，保证段序 = 播放序 = 水印码位序

// segNum 从文件名里抽出段号；抽不出返回 -1
func segNum(path string) int {
	base := filepath.Base(path)
	// 去掉扩展名后，取最后一段连续数字
	base = strings.TrimSuffix(base, filepath.Ext(base))
	digits := ""
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] >= '0' && base[i] <= '9' {
			digits = string(base[i]) + digits
		} else if digits != "" {
			break
		}
	}
	if digits == "" {
		return -1
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return -1
	}
	return n
}

// sortSegmentPaths 按段号数值升序就地排序
func sortSegmentPaths(paths []string) {
	sort.SliceStable(paths, func(i, j int) bool {
		return segNum(paths[i]) < segNum(paths[j])
	})
}
