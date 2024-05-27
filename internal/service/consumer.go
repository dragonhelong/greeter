package service

import (
	"context"
	"fmt"
	"log"

	"github.com/loonghe/grpc_greeter_helloworld/internal/repo/db"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/config"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"

	"github.com/segmentio/kafka-go"
)

// Handle consumer消费者，通过Reader接收消息
func Handle(ctx context.Context) {
	// 创建Reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{config.Viper.GetString("kafka.broker")},
		Topic:     config.Viper.GetString("kafka.topic"),
		Partition: 0,
		MaxBytes:  10e6, // 10MB
	})
	r.SetOffset(0) // 设置Offset

	// 接收消息
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		if err := NewConsumer(db.NewMysql()).consumerAction(ctx, m); err != nil {
			zaplog.Sugar.Errorf("consumer: consumerAction err: %v", err)
		}
	}

	// 程序退出前关闭Reader
	if err := r.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}
}

type consumerImpl struct {
	store db.Registry
}

func NewConsumer(store db.Registry) *consumerImpl {
	return &consumerImpl{store: store}
}

func (c *consumerImpl) consumerAction(ctx context.Context, m kafka.Message) error {
	user, _ := c.store.UserStore(ctx).GetUser(ctx, 1)
	fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
	fmt.Printf("user: %v\n", user)
	return nil
}
