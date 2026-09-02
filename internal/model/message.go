package model

import (
	"time"

	"gorm.io/gorm"
)

// Message 是用户提交给管理员处理的留言。
// 留言创建后保持只读，普通用户只能通过用户维度查询自己的记录。
type Message struct {
	ID         uint           `gorm:"primaryKey;autoIncrement;index:idx_messages_user_deleted_created,priority:4;index:idx_messages_deleted_created,priority:3;comment:主键ID" json:"id"`
	CreateTime time.Time      `gorm:"column:create_time;autoCreateTime;index:idx_messages_user_deleted_created,priority:3;index:idx_messages_deleted_created,priority:2;comment:创建时间" json:"create_time"`
	UpdateTime time.Time      `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"update_time"`
	Deleted    gorm.DeletedAt `gorm:"column:deleted;index:idx_messages_user_deleted_created,priority:2;index:idx_messages_deleted_created,priority:1;comment:删除时间，空表示未删除" json:"-"`
	UserID     uint           `gorm:"not null;index:idx_messages_user_deleted_created,priority:1;comment:留言用户ID" json:"user_id"`
	Content    string         `gorm:"type:text;not null;comment:留言内容" json:"content"`
	User       *User          `gorm:"-" json:"user"`
}

func (Message) TableComment() string {
	return "用户留言表"
}
