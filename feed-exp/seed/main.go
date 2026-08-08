package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}

func main() {
	var (
		dsn           = flag.String("mysql-dsn", "gorm:gorm12345@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local", "MySQL DSN")
		redisAddr     = flag.String("redis-addr", "localhost:6379", "Redis 地址")
		users         = flag.Int("users", 10000, "用户数")
		videos        = flag.Int("videos", 50000, "视频数")
		bigvFans      = flag.Int("bigv-fans", 0, "大V粉丝数（模拟写扩散压测，0=不生成）")
		followsPerUser = flag.Int("follows-per-user", 30, "每用户关注数")
		feedLimit     = flag.Int("feed-limit", 200, "每用户 feed 上限")
		tokens        = flag.Int("tokens", 2000, "签发 token 数")
		tokenFile     = flag.String("token-file", "../tokens.txt", "token 输出文件")
		jwtSecret     = flag.String("jwt-secret", "fONAdtaV+MOx4kBtsPgdtkJRiHqpu6Z0H+Jog4JOjn4=", "JWT 密钥（须与服务器 config.JWTSecret 一致）")
		reset         = flag.Bool("reset", false, "清空六表 + FLUSHDB Redis")
		feedMySQL     = flag.Bool("feed-mysql", true, "写退化版 MySQL feed 表")
		feedRedis     = flag.Bool("feed-redis", true, "写优化版 Redis feed")
		countsRedis   = flag.Bool("counts-redis", true, "写优化版 Redis 计数 hash（与 MySQL 列一致）")
		batch         = flag.Int("batch", 1000, "每批 INSERT 行数")
		randomSeed    = flag.Int64("random-seed", 42, "随机种子（保证数据可复现）")
	)
	flag.Parse()

	db, err := connectMySQL(*dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	rdb, err := connectRedis(*redisAddr)
	if err != nil {
		fatal(err)
	}
	defer rdb.Close()

	// 建表（幂等）+ 可选重置
	if err := createTables(db); err != nil {
		fatal(err)
	}
	if *reset {
		if err := truncateTables(db); err != nil {
			fatal(err)
		}
		if err := rdb.FlushDB(context.Background()).Err(); err != nil {
			fatal(err)
		}
		fmt.Println("已重置：六表 TRUNCATE + Redis FLUSHDB")
	}

	now := time.Now()
	rng := rand.New(rand.NewSource(*randomSeed))

	// 统一密码 123456 的 bcrypt 哈希（供登录验证）
	pwdHash, err := bcrypt.GenerateFromPassword([]byte("123456"), 10)
	if err != nil {
		fatal(err)
	}

	if err := insertUsers(db, *users, *batch, string(pwdHash), now); err != nil {
		fatal(err)
	}

	// 大 V 写扩散数据（粉丝 + 关注大 V + 大 V/粉丝 token）
	if *bigvFans > 0 {
		info := insertBigV(db, *users, *bigvFans, *batch, string(pwdHash), now)
		var sb strings.Builder
		bt, err := signToken(info.BigVID, *jwtSecret)
		if err != nil {
			fatal(err)
		}
		sb.WriteString(fmt.Sprintf("bigv:%d %s\n", info.BigVID, bt))
		// 前 3 个粉丝 token，供 fanout_test.sh 轮询 feed 检测扩散完成
		for i := 0; i < 3 && i < info.FanCount; i++ {
			ft, err := signToken(info.FanStart+int64(i), *jwtSecret)
			if err != nil {
				fatal(err)
			}
			sb.WriteString(fmt.Sprintf("fan:%d %s\n", info.FanStart+int64(i), ft))
		}
		if err := os.WriteFile("../bigv.txt", []byte(sb.String()), 0o644); err != nil {
			fatal(err)
		}
		fmt.Printf("大V 信息 → ../bigv.txt（bigv:%d + 前3粉丝 token）\n", info.BigVID)
	}

	infos, byUploader := insertVideos(db, *videos, *users, *batch, rng, now)
	follows := insertFollows(db, *users, *followsPerUser, *batch, rng, now)

	// feed 对称初始化：同一份关注关系/视频，同时写两版存储
	feeds := computeFeeds(follows, byUploader, *feedLimit, *users)
	if *feedMySQL {
		if err := writeFeedMySQL(db, feeds, *batch, now); err != nil {
			fatal(err)
		}
	}
	if *feedRedis {
		if err := writeFeedRedis(rdb, feeds); err != nil {
			fatal(err)
		}
	}

	// Redis 计数一致化（优化版读 Redis hash，退化版读 MySQL 列，两版数值必须相同）
	if *countsRedis {
		if err := writeCountsRedis(rdb, infos, 1000); err != nil {
			fatal(err)
		}
	}

	// 签发 JWT token 池
	if err := signTokens(*tokens, *jwtSecret, *tokenFile); err != nil {
		fatal(err)
	}

	fmt.Println("灌数据完成 ✓")
}
