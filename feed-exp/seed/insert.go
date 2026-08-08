package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// videoInfo 视频基本信息（供 Redis 计数一致化用）
type videoInfo struct {
	id   int64
	play int64
	like int64
	fav  int64
}

// execBatch 多行 VALUES 批量插入：每 batch 行一次 Exec
func execBatch(db *sql.DB, query string, batch int, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += batch {
		end := i + batch
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]
		ph := make([]string, len(chunk))
		for j := range chunk {
			ph[j] = "(" + strings.Repeat("?,", len(chunk[j])-1) + "?)"
		}
		full := query + strings.Join(ph, ",")
		args := make([]any, 0, len(chunk)*len(chunk[0]))
		for _, r := range chunk {
			args = append(args, r...)
		}
		if _, err := db.Exec(full, args...); err != nil {
			return fmt.Errorf("批量插入失败（第 %d 批）: %w", i/batch, err)
		}
	}
	return nil
}

// insertUsers 灌用户：唯一 email、统一密码哈希、假头像路径
func insertUsers(db *sql.DB, n, batch int, pwdHash string, now time.Time) error {
	fmt.Printf("灌入 %d 用户...\n", n)
	rows := make([][]any, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, []any{
			fmt.Sprintf("用户%d", i), // name
			fmt.Sprintf("u%d@example.com", i), // email（唯一）
			nil,       // p_number
			pwdHash,   // password（bcrypt("123456")）
			fmt.Sprintf("/uploads/avatar/a%d.jpg", i), // avatar（假路径）
			"",        // bio
			now,       // created_at
			now,       // updated_at
		})
	}
	base := "INSERT INTO account (name, email, p_number, password, avatar, bio, created_at, updated_at) VALUES "
	return execBatch(db, base, batch, rows)
}

// insertVideos 灌视频：发布者均匀随机、假 URL（不落盘媒体）、随机初始计数
// 返回视频信息列表 + 发布者→视频id映射（升序）
func insertVideos(db *sql.DB, n, nUsers, batch int, rng *rand.Rand, now time.Time) ([]videoInfo, map[int64][]int64) {
	fmt.Printf("灌入 %d 视频...\n", n)
	infos := make([]videoInfo, 0, n)
	byUploader := make(map[int64][]int64)
	rows := make([][]any, 0, n)
	for i := 1; i <= n; i++ {
		up := int64(rng.Intn(nUsers)) + 1
		p := int64(100 + rng.Intn(99901)) // 100..100000
		l := int64(rng.Intn(1001))        // 0..1000
		f := int64(rng.Intn(501))         // 0..500
		rows = append(rows, []any{
			fmt.Sprintf("测试视频%d", i),    // title
			fmt.Sprintf("/uploads/video/cover/c%d.jpg", i), // cover_url
			fmt.Sprintf("/uploads/video/v%d.mp4", i),       // video_url（假路径）
			"",          // describe
			up,          // uploader
			int64(10 + rng.Intn(290)), // duration 10..299s
			p, l, f,     // play/like/fav
			1,           // status
			now,         // created_at
			now,         // updated_at
		})
		infos = append(infos, videoInfo{id: int64(i), play: p, like: l, fav: f})
		byUploader[up] = append(byUploader[up], int64(i))
	}
	base := "INSERT INTO video (title, cover_url, video_url, `describe`, uploader, duration, play_count, like_count, fav_count, status, created_at, updated_at) VALUES "
	if err := execBatch(db, base, batch, rows); err != nil {
		panic(err)
	}
	return infos, byUploader
}

// insertFollows 灌关注关系：每用户随机 ~perUser 个不重复目标（排除自己）
// 返回 用户→关注列表
func insertFollows(db *sql.DB, nUsers, perUser, batch int, rng *rand.Rand, now time.Time) map[int64][]int64 {
	fmt.Printf("灌入关注关系（每用户 ~%d 个，共约 %d 条）...\n", perUser, nUsers*perUser)
	follows := make(map[int64][]int64, nUsers)
	rows := make([][]any, 0, nUsers*perUser)
	for u := 1; u <= nUsers; u++ {
		seen := make(map[int64]struct{}, perUser)
		targets := make([]int64, 0, perUser)
		for len(targets) < perUser {
			t := int64(rng.Intn(nUsers)) + 1
			if t == int64(u) {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			targets = append(targets, t)
		}
		follows[int64(u)] = targets
		for _, t := range targets {
			rows = append(rows, []any{int64(u), t, now})
		}
	}
	base := "INSERT INTO follows (follower_id, followee_id, created_at) VALUES "
	if err := execBatch(db, base, batch, rows); err != nil {
		panic(err)
	}
	return follows
}

// bigVInfo 大 V 压测数据（fanout_test.sh 读取）
type bigVInfo struct {
	BigVID    int64
	FanStart  int64
	FanCount  int
	BigVToken string
}

// insertBigV 生成 bigvFans 个真实粉丝账号 + 1 个大 V 账号，每个粉丝关注大 V
// 粉丝 id 从 baseUsers+1 起；大 V id = baseUsers+bigvFans+1
// 返回大 V 信息（供写 bigv.txt）
func insertBigV(db *sql.DB, baseUsers, bigvFans, batch int, pwdHash string, now time.Time) *bigVInfo {
	fmt.Printf("灌入大V：%d 个真实粉丝账号 + 1 个大V...\n", bigvFans)
	fanStart := int64(baseUsers) + 1
	bigvID := fanStart + int64(bigvFans)

	// 1) account：粉丝 + 大 V
	rows := make([][]any, 0, bigvFans+1)
	for i := int64(0); i < int64(bigvFans); i++ {
		uid := fanStart + i
		rows = append(rows, []any{
			fmt.Sprintf("粉丝%d", uid),            // name
			fmt.Sprintf("fan%d@example.com", uid), // email（唯一）
			nil, pwdHash,
			fmt.Sprintf("/uploads/avatar/f%d.jpg", uid), // avatar
			"", now, now,
		})
	}
	rows = append(rows, []any{
		"大V博主",       // name
		"bigv@example.com", // email
		nil, pwdHash,
		"/uploads/avatar/bigv.jpg",
		"", now, now,
	})
	base := "INSERT INTO account (name, email, p_number, password, avatar, bio, created_at, updated_at) VALUES "
	if err := execBatch(db, base, batch, rows); err != nil {
		panic(err)
	}

	// 2) follows：每个粉丝关注大 V
	fmt.Printf("灌入 %d 条 粉丝→大V 关注...\n", bigvFans)
	fRows := make([][]any, 0, bigvFans)
	for i := int64(0); i < int64(bigvFans); i++ {
		fRows = append(fRows, []any{fanStart + i, bigvID, now})
	}
	baseF := "INSERT INTO follows (follower_id, followee_id, created_at) VALUES "
	if err := execBatch(db, baseF, batch, fRows); err != nil {
		panic(err)
	}

	fmt.Printf("大V id=%d，粉丝 id=%d..%d\n", bigvID, fanStart, fanStart+int64(bigvFans)-1)
	return &bigVInfo{BigVID: bigvID, FanStart: fanStart, FanCount: bigvFans}
}
