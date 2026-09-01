package model

import "time"

// PostPinDuration 是管理员可选择的置顶期限。
type PostPinDuration string

const (
	PostPinDurationDay       PostPinDuration = "day"
	PostPinDurationWeek      PostPinDuration = "week"
	PostPinDurationMonth     PostPinDuration = "month"
	PostPinDurationPermanent PostPinDuration = "permanent"
)

// PermanentPostPinnedUntil 使用 MariaDB DATETIME(3) 可表示的最大时间作为永久置顶标记。
// 这能兼容只认识 pinned_until 的旧版本，无需新增数据库字段。
var PermanentPostPinnedUntil = time.Date(9999, time.December, 31, 23, 59, 59, 999000000, time.Local)

func IsValidPostPinDuration(duration PostPinDuration) bool {
	switch duration {
	case PostPinDurationDay, PostPinDurationWeek, PostPinDurationMonth, PostPinDurationPermanent:
		return true
	default:
		return false
	}
}

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

// IsPermanentlyPinned 判断 pinned_until 是否为永久置顶哨兵。
// DATETIME 不保存时区，所以按墙钟字段和毫秒精度判断。
func (p *Post) IsPermanentlyPinned() bool {
	if p == nil || p.PinnedUntil == nil {
		return false
	}
	value := p.PinnedUntil
	return value.Year() == 9999 &&
		value.Month() == time.December &&
		value.Day() == 31 &&
		value.Hour() == 23 &&
		value.Minute() == 59 &&
		value.Second() == 59 &&
		value.Nanosecond()/int(time.Millisecond) == 999
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
