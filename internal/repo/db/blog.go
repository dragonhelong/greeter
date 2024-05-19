package db

import (
	"context"
	"fmt"

	"github.com/loonghe/grpc_greeter_helloworld/internal/model"

	"gorm.io/gorm"
)

// BlogStore defines db operations of blog
type BlogStore interface {
	GetBlog(ctx context.Context, id uint64) (*model.Blog, error)
}

// blogStoreImpl defines blog Store implementation
type blogStoreImpl struct {
	db *gorm.DB
}

var _ BlogStore = (*blogStoreImpl)(nil)

// NewBlogStore creates new blog Store
func NewBlogStore(db *gorm.DB) BlogStore {
	return &blogStoreImpl{db: db}
}

// GetBlog gets blog detail from the storage.
func (b *blogStoreImpl) GetBlog(ctx context.Context, id uint64) (*model.Blog, error) {
	var blog *model.Blog
	if err := b.db.WithContext(ctx).
		Model(&model.Blog{}).
		Where("id = ?", id).
		Find(&blog).Error; err != nil {
		fmt.Printf("get blog err: %v", err)
		return nil, err
	}
	return blog, nil
}
