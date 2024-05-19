// Package db defines database operations of user and blog.
package db

import (
	"context"
	"fmt"
	"runtime"

	"github.com/loonghe/grpc_greeter_helloworld/pkg/config"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"

	driver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Registry defines db store related interface registry
type Registry interface {
	BlogStore(ctx context.Context) BlogStore
	UserStore(ctx context.Context) UserStore
	Transaction(ctx context.Context, fn func(ds Registry) error) error
}

// mysql finish Registry interface
type mysql struct {
	db *gorm.DB
}

var _ Registry = (*mysql)(nil)

// NewMysql create new mysql connection pool and db store registry
func NewMysql() Registry {
	db, err := gorm.Open(driver.Open(config.Viper.GetString("mysql.dsn")), &gorm.Config{})
	if err != nil {
		zaplog.Sugar.Errorf("create mysql connection pool err: %v", err)
	}
	return &mysql{db: db}
}

/* struct mysql finish Registry interface */
// BlogStore gets blog store
func (m *mysql) BlogStore(ctx context.Context) BlogStore {
	return NewBlogStore(m.db)
}

// BlogStore gets user store
func (m *mysql) UserStore(ctx context.Context) UserStore {
	return NewUserStore(m.db)
}

// Transaction implements a wrapper of transaction logic in repository layer,
// which could be used in logic layer to combine business logic and repo logic in need.
func (m *mysql) Transaction(ctx context.Context, fn func(ds Registry) error) error {
	var err error
	gx := m.db.WithContext(ctx).Begin()
	defer func() {
		if p := recover(); p != nil {
			gx.Rollback()
			switch e := p.(type) {
			case runtime.Error:
				panic(e)
			case error:
				err = fmt.Errorf("panic err: %v", p)
				return
			default:
				panic(e)
			}
		}
		if err != nil {
			if rbErr := gx.Rollback(); rbErr != nil {
				err = fmt.Errorf("gx err: %v, rb err:  %v", err, rbErr)
			}
		} else {
			gx.Commit()
		}
	}()
	newDB := &mysql{db: gx}
	err = fn(newDB)
	return err
}
