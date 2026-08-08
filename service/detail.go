package service

import (
	"context"
	"encoding/json"
	"errors"
	"feed/dao"
	"feed/model"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrVideoNotFound = errors.New("video not found")

const (
	videoStaticCachePrefix = "video:staticCache:"
	videoCountCachePrefix  = "video:countCache:"
	baseTime               = time.Hour
	// emptyCacheValue 缓存空值哨兵：标记"该视频不存在"，挡缓存穿透
	emptyCacheValue = "__NOT_EXIST__"
	// emptyCacheTTL 空值缓存短 TTL：不长期占位，避免挡住之后真实创建的视频
	emptyCacheTTL = 5 * time.Minute
)

// getCache 读详情缓存：命中哨兵=视频不存在，命中数据=返回，miss=继续查库
func getCache(videoID int64) (*model.Video, error) {
	key := videoStaticCachePrefix + strconv.FormatInt(videoID, 10)
	cached, err := dao.RDB.Get(context.Background(), key).Result()
	if err == nil {
		if cached == emptyCacheValue { // 命中空值哨兵 → 视频不存在，直接返回，不再查库
			return nil, ErrVideoNotFound
		}
		var video model.Video
		if err := json.Unmarshal([]byte(cached), &video); err == nil {
			return &video, nil
		}
		dao.RDB.Del(context.Background(), key)
	} else if err != redis.Nil {
		return nil, err
	}
	return nil, nil // redis.Nil = 正常缓存 miss，继续查库
}

// setCache 写详情缓存：随机 TTL 防雪崩（1小时 ± 5分钟）
func setCache(videoID int64, video *model.Video) error {
	key := videoStaticCachePrefix + strconv.FormatInt(videoID, 10)
	second := rand.Intn(2*300+1) - 300
	data, err := json.Marshal(video)
	if err != nil {
		return err
	}
	if _, err := dao.RDB.Set(context.Background(), key, data, time.Duration(second)*time.Second+baseTime).Result(); err != nil {
		return err
	}
	return nil
}

// setEmptyCache 写入空值缓存：查库确认视频不存在后调用，挡缓存穿透；短 TTL 防止挡住后续真实创建
func setEmptyCache(videoID int64) {
	key := videoStaticCachePrefix + strconv.FormatInt(videoID, 10)
	dao.RDB.Set(context.Background(), key, emptyCacheValue, emptyCacheTTL)
}

// delCache 删除视频详情缓存：写操作（创建/删除视频）后调用，保证缓存与数据库一致
func delCache(videoID int64) {
	key := videoStaticCachePrefix + strconv.FormatInt(videoID, 10)
	dao.RDB.Del(context.Background(), key)
}

// fillCount 从 Redis 计数 hash 填充播放/点赞/收藏数（Redis 挂了静默保留 MySQL 兜底）
func fillCount(video *model.Video) {
	key := "video:count:" + strconv.FormatInt(video.ID, 10)
	res, err := dao.RDB.HGetAll(context.Background(), key).Result()
	if err != nil {
		return // Redis 挂了：静默保留 MySQL 兜底值
	}
	if play, ok := res["play"]; ok {
		video.PlayCount, _ = strconv.ParseInt(play, 10, 64)
	}
	if like, ok := res["like"]; ok {
		video.LikeCount, _ = strconv.ParseInt(like, 10, 64)
	}
	if fav, ok := res["fav"]; ok {
		video.FavCount, _ = strconv.ParseInt(fav, 10, 64)
	}
}

// ifGetCached 读缓存并填计数：命中返回视频，miss 返回 nil
func ifGetCached(videoID int64) (*model.Video, error) {
	cached, err := getCache(videoID)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		fillCount(cached)
		return cached, nil
	}
	return nil, nil
}

// GetDetail 传入视频ID，返回info结构体,err
func GetDetail(videoID int64) (*model.Video, error) {
	if cached, err := ifGetCached(videoID); err != nil {
		return nil, err
	} else if cached != nil {
		return cached, nil
	}

	// 未命中情况
	uuid := generateRandomString(9) // 假设最多10000个请求打过来，概率大概是1/270m
	for count := 10; count > 0; count-- {
		if ok, _ := dao.RDB.SetNX(context.Background(), "lock:"+strconv.FormatInt(videoID, 10),
			"lockedBy:"+uuid, 3*time.Second).Result(); ok {
			if cached, err := ifGetCached(videoID); err != nil { // dounble check
				return nil, err
			} else if cached != nil {
				dao.RDB.Del(context.Background(), "lock:"+strconv.FormatInt(videoID, 10))
				return cached, nil
			}
			break
		}
		time.Sleep(500 * time.Millisecond)
		// 抢锁失败：先查缓存，别人可能已建好，命中直接返回（避免排队抢锁）
		if cached, err := ifGetCached(videoID); err != nil {
			return nil, err
		} else if cached != nil {
			return cached, nil
		}
	}
	// 兜底查缓存
	if cached, err := ifGetCached(videoID); err != nil {
		return nil, err
	} else if cached != nil {
		return cached, nil
	}

	video, err := dao.FindByVideoID(videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		setEmptyCache(videoID) // 查库确认不存在 → 写空值缓存，挡后续穿透
		return nil, ErrVideoNotFound
	}

	var id int64
	var name, avatar string
	account, err := dao.FindByID(video.Uploader)
	if err != nil {
		return nil, err
	}
	if account == nil {
		name = "已注销的用户"
		avatar = "" // 约定：avatar 未设置为空字符串，前端显示占位
	} else {
		id = account.ID
		name = account.Name
		avatar = account.Avatar
	}

	publisher := model.PublisherInfo{ID: id, Name: name, Avatar: avatar}
	video.Publisher = publisher

	if err := setCache(videoID, video); err != nil {
		return nil, err
	}
	if err := dao.RDB.Eval(context.Background(), unlockScript,
		[]string{"lock:" + strconv.FormatInt(videoID, 10)}, "lockedBy:"+uuid).Err(); err != nil {
		return nil, err
	}
	fillCount(video)
	return video, nil
}
