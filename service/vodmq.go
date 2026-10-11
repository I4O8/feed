package service

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 点播模块：打包任务队列
// 文档：docs/点播模块.md 4.3 —— 挂在已有 RabbitMQ 上做打包任务队列，复用 feed 模块的 MQ 经验
// 上传源片 → 立即返回 → 丢消息 → worker 消费打包 → 回写状态；前端轮询 status 看进度

// vodChannel 打包队列专用 channel（与 feed 的 fanout 用不同 channel，互不干扰）
var vodChannel *amqp.Channel

// PackTask 打包任务消息
type PackTask struct {
	MovieID int64 `json:"movieId"`
}

// InitVodMQ 声明打包队列；需要 feed MQ 已连接（复用同一 amqp 连接）
func InitVodMQ() error {
	if conn == nil {
		return fmt.Errorf("RabbitMQ 未连接，请先 InitFeedMQ()")
	}
	var err error
	vodChannel, err = conn.Channel()
	if err != nil {
		return err
	}
	// 打包队列：durable + 手动 ack；失败重试交给 worker 内部（打包幂等，可重跑）
	if _, err := vodChannel.QueueDeclare("vod_pack", true, false, false, false, nil); err != nil {
		return err
	}
	return nil
}

// PublishPackTask 投递打包任务（上传源片后调用，立即返回）
func PublishPackTask(movieID int64) error {
	if vodChannel == nil {
		return fmt.Errorf("打包队列未初始化")
	}
	body, err := json.Marshal(PackTask{MovieID: movieID})
	if err != nil {
		return err
	}
	return vodChannel.PublishWithContext(context.Background(), "", "vod_pack", false, false,
		amqp.Publishing{Body: body, DeliveryMode: amqp.Persistent})
}

// StartPackConsumer 常驻消费打包任务
// 与 feed 的消费者同一套写法：坏消息丢弃、失败按重试次数决定是否 re-queue
func StartPackConsumer() {
	if vodChannel == nil {
		fmt.Println("打包队列未初始化，跳过消费者")
		return
	}
	if err := vodChannel.Qos(1, 0, false); err != nil { // 打包重（CPU/IO），一次只取一个
		fmt.Printf("设置 Qos 失败: %v\n", err)
	}
	msgs, err := vodChannel.Consume("vod_pack", "", false, false, false, false, nil)
	if err != nil {
		fmt.Printf("打包队列消费失败: %v\n", err)
		return
	}
	go func() {
		for msg := range msgs {
			var task PackTask
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				fmt.Printf("打包任务解析失败: %v\n", err)
				msg.Ack(false)
				continue
			}
			if err := PackMovie(task.MovieID); err != nil {
				fmt.Printf("影片 %d 打包失败: %v\n", task.MovieID, err)
				// 失败已写进 movie.status/fail_reason，不 re-queue（避免毒消息死循环）
				msg.Ack(false)
				continue
			}
			msg.Ack(false)
		}
	}()
}
