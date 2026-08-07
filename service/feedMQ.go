package service

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// amqp API 用法参考：mq-practice/main.go

var (
	conn *amqp.Connection
	ch   *amqp.Channel
)

type vidAndUid struct {
	VideoID    int64 `json:"videoId"`
	UploaderID int64 `json:"uploaderId"`
}

// InitFeedMQ 连接 RabbitMQ + 声明队列 feedpush（durable）+ 存全局 conn/ch
func InitFeedMQ() error {
	var err error
	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}

	// chanal创建
	ch, err = conn.Channel()
	if err != nil {
		return err
	}

	// 声明
	if err := ch.ExchangeDeclare("fanout_main", "direct",
		true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("fanout_delay", "direct",
		true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("feedpush", true, false,
		false, false, amqp.Table{"x-dead-letter-exchange": "fanout_delay"}); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("delay_queue", true, false, false,
		false, amqp.Table{"x-message-ttl": 60000, "x-dead-letter-exchange": "fanout_main"}); err != nil {
		return err
	}
	if err := ch.QueueBind("feedpush", "feedpush", "fanout_main",
		false, nil); err != nil {
		return err
	}
	// delay_queue 用 feedpush 做路由键：死信消息到 fanout_delay 时带的是原始 routingKey=feedpush
	if err := ch.QueueBind("delay_queue", "feedpush", "fanout_delay",
		false, nil); err != nil {
		return err
	}

	return nil
}

// PublishFanoutTask 发扩散任务
func PublishFanoutTask(videoID, uploaderID int64) error {
	task := vidAndUid{videoID, uploaderID}
	jsonBytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(context.Background(), "fanout_main", "feedpush", false,
		false, amqp.Publishing{Body: jsonBytes, DeliveryMode: amqp.Persistent})
}

// StartFanoutConsumer 消费者：常驻 goroutine，取消息 → fanout() → Ack
func StartFanoutConsumer() {
	msgs, err := ch.Consume("feedpush",
		"", false, false, false, false, nil)
	if err != nil {
		fmt.Printf("消费失败: %v", err)
		return
	}
	go func() {
		for msg := range msgs {
			var tmp vidAndUid
			if err := json.Unmarshal(msg.Body, &tmp); err != nil {
				fmt.Printf("解析失败: %v\n", err)
				msg.Ack(false) // 坏消息，直接丢弃
				continue
			}

			// 成功
			if err := fanout(tmp.VideoID, tmp.UploaderID); err == nil {
				msg.Ack(false)
				continue
			}

			// 失败：数死信次数
			deaths, _ := msg.Headers["x-death"].([]interface{})
			fmt.Printf("fanout 失败(第%d轮延迟): %v\n", len(deaths)+1, err)
			if len(deaths) < 3 {
				msg.Nack(false, false) // 进延迟队列，TTL 后回投再试
			} else {
				msg.Ack(false) // 3 轮用完，最终放弃
			}
		}
	}()
}
