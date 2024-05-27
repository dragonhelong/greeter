// Package producer define message push of user, blog for service
package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loonghe/grpc_greeter_helloworld/internal/model"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"

	"github.com/segmentio/kafka-go"
)

// UserEvent 定义 user 事件消息生产操作接口
type UserEvent interface {
	Emit(ctx context.Context, event *model.User) error
}

type userEventImpl struct {
	writer kafka.Writer
}

// NewUserEvent 创建 UserEvent 接口实现
func NewUserEvent(addr, topic string) UserEvent {
	return &userEventImpl{
		writer: kafka.Writer{
			Addr:                   kafka.TCP(addr),
			Topic:                  topic,
			RequiredAcks:           kafka.RequireAll,
			Async:                  true,
			AllowAutoTopicCreation: true,
		},
	}
}

// Emit 发出 user 事件消息
func (b *userEventImpl) Emit(ctx context.Context, event *model.User) error {
	message, err := json.Marshal(event)
	if err != nil {
		zaplog.Sugar.Errorf("user 事件解析失败: %v", err)
		return err
	}
	if err := b.writer.WriteMessages(ctx, kafka.Message{Value: message}); err != nil {
		return fmt.Errorf("user 事件发送失败 err: %v, event: %v", err, message)
	}
	return nil
}
