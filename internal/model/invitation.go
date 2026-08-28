package model

import "time"

// Invitation 是用户创建的一次性注册邀请码。
// Code 需要在注册请求中原样提交；UsedBy/UsedAt 同时为空表示尚未使用。
type Invitation struct {
	ID         uint       `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"-"`
	CreateTime time.Time  `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"-"`
	UpdateTime time.Time  `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"-"`
	Code       string     `gorm:"type:char(6);not null;uniqueIndex;comment:邀请码" json:"-"`
	CreatorID  uint       `gorm:"column:creator_id;not null;index;comment:创建用户ID" json:"-"`
	UsedBy     *uint      `gorm:"column:used_by;uniqueIndex;comment:使用用户ID" json:"-"`
	UsedAt     *time.Time `gorm:"column:used_at;comment:使用时间" json:"-"`
}

func (Invitation) TableName() string {
	return "invitations"
}

func (Invitation) TableComment() string {
	return "注册邀请码表"
}
