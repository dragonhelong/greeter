// Package model defines domain entities and database tables' related struts
package model

// BaseModel includes ID, CreatedAt, UpdatedAt and DeletedAt.
type BaseModel struct {
	ID        uint64 `gorm:"primaryKey;column:id;type:bigint(20) unsigned;not null"  json:"id"`
	CreatedAt int64  `gorm:"column:created_at;autoCreateTime"                 json:"created_at"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime"                 json:"updated_at"`
	DeletedAt int64  `gorm:"column:deleted_at"                                json:"deleted_at"`
}
