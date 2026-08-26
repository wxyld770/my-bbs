package model

import "time"

// 帖子可见性
const (
	VisiblePrivate uint8 = 0 // 仅自己可见
	VisiblePublic  uint8 = 1 // 所有人可见
)

// IsValidVisible 校验可见性取值是否合法；新增类型时只改这里
func IsValidVisible(visible uint8) bool {
	switch visible {
	case VisiblePrivate, VisiblePublic:
		return true
	default:
		return false
	}
}

// IsPrivate 是否仅自己可见
func (p *Post) IsPrivate() bool {
	return p.Visible == VisiblePrivate
}

// IsPinnedAt 判断帖子在指定时刻是否处于有效置顶期。
func (p *Post) IsPinnedAt(now time.Time) bool {
	return p != nil && p.PinnedUntil != nil && p.PinnedUntil.After(now)
}

// Post 帖子模型
type Post struct {
	BaseModel
	UserID  uint   `gorm:"index;not null" json:"user_id"`
	Title   string `gorm:"type:varchar(255);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`
	Visible uint8  `gorm:"type:tinyint(4);not null;comment:可见性状态，1所有人可见，0仅自己可见" json:"visible"`
	// PinnedUntil 为 nil 或不晚于当前时间时，帖子不处于有效置顶状态。
	PinnedUntil *time.Time `gorm:"column:pinned_until;index;comment:置顶到期时间" json:"pinned_until"`
	User        *User      `json:"user" gorm:"-"` // 非 ORM 关联，由 Service 手动填充
}

func (Post) TableComment() string {
	return "帖子信息表"
}
