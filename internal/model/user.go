package model

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
