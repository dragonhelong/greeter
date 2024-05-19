package db

import (
	"context"
	"fmt"

	"github.com/loonghe/grpc_greeter_helloworld/internal/model"

	"gorm.io/gorm"
)

// UserStore defines db operations of user
type UserStore interface {
	GetUser(ctx context.Context, id uint64) (*model.User, error)
}

// userStoreImpl defines user Store implementation
type userStoreImpl struct {
	db *gorm.DB
}

var _ UserStore = (*userStoreImpl)(nil)

// NewUserStore creates new user Store
func NewUserStore(db *gorm.DB) UserStore {
	return &userStoreImpl{db: db}
}

// GetUser gets user detail from the storage.
func (u *userStoreImpl) GetUser(ctx context.Context, id uint64) (*model.User, error) {
	var user *model.User
	if err := u.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Find(&user).Error; err != nil {
		fmt.Printf("get user err: %v", err)
		return nil, err
	}
	return user, nil
}
