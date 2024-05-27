// Package logic define use case operations of user, blog for service
package logic

import (
	"context"

	"github.com/loonghe/grpc_greeter_helloworld/internal/model"
	"github.com/loonghe/grpc_greeter_helloworld/internal/repo/db"
	"github.com/loonghe/grpc_greeter_helloworld/internal/repo/producer"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
)

// UserUseCase defines user use case interface.
type UserUseCase interface {
	GetUser(ctx context.Context, id uint64) (*model.User, error)
}

// userUseCaseImpl defines user use case implementation.
type userUseCaseImpl struct {
	store db.Registry
	event producer.UserEvent
}

var _ UserUseCase = (*userUseCaseImpl)(nil)

// NewUserUseCase creates new user use case.
func NewUserUseCase(store db.Registry, event producer.UserEvent) UserUseCase {
	return &userUseCaseImpl{store: store, event: event}
}

// GetUser gets user detail.
func (u *userUseCaseImpl) GetUser(ctx context.Context, id uint64) (*model.User, error) {
	user, err := u.store.UserStore(ctx).GetUser(ctx, id)
	if err != nil {
		zaplog.Sugar.Errorf("logic: get user detail err: %v", err)
		return nil, err
	}
	// 测试调用producer生产消息
	if err := u.event.Emit(ctx, user); err != nil {
		zaplog.Sugar.Errorf("logic: producer emit user detail err: %v", err)
	}
	return user, nil
}
