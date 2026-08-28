package model

import "time"

// 用户状态
const (
	UserStatusMuted  uint = 0 // 禁言
	UserStatusNormal uint = 1 // 正常
)

// User 用户模型
type User struct {
	BaseModel
	Username     string `gorm:"type:varchar(64);not null;uniqueIndex;comment:用户名" json:"username"`
	Password     string `gorm:"type:varchar(255);not null" json:"-"` // 不返回给前端
	Nickname     string `gorm:"type:varchar(64);comment:用户昵称" json:"nickname"`
	Status       uint   `gorm:"type:tinyint(4);default:1;comment:用户状态，1:正常，0禁言" json:"status"`
	Introduction string `gorm:"type:varchar(1024);comment:个人介绍" json:"introduction"`
	// AvatarURL 在生产 full_ddl.sql 中使用 utf8mb4_bin，保留 URL 路径的大小写语义。
	AvatarURL string `gorm:"column:avatar_url;type:varchar(2048);comment:头像图片链接" json:"avatar_url"`
	// AvatarUpdatedAt 独立记录头像修改时间，避免昵称/简介更新占用头像修改额度。
	AvatarUpdatedAt *time.Time `gorm:"column:avatar_updated_at;comment:头像最后修改时间" json:"-"`
	// InviteCode 记录注册该账号时消费的邀请码。历史用户没有邀请码，值为 NULL。
	InviteCode *string `gorm:"column:invite_code;type:char(6);uniqueIndex;comment:注册时使用的邀请码" json:"-"`
}

// IsActive 是否可登录 / 访问需登录接口
func (u *User) IsActive() bool {
	return u != nil && u.Status == UserStatusNormal
}

func (User) TableComment() string {
	return "用户信息表"
}
