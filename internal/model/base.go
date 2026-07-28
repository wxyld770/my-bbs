package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 手写公共字段，替代 gorm.Model
type BaseModel struct {
	ID         uint           `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	CreateTime time.Time      `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"create_time"`
	UpdateTime time.Time      `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"update_time"`
	Deleted    gorm.DeletedAt `gorm:"column:deleted;index;comment:删除时间，空表示未删除" json:"-"`
}
