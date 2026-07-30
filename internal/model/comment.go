package model

// Comment 评论模型（扁平评论，挂在帖子下）
type Comment struct {
	BaseModel
	PostID  uint   `gorm:"index;not null" json:"post_id"`
	UserID  uint   `gorm:"index;not null" json:"user_id"`
	Content string `gorm:"type:text;not null" json:"content"`
	User    *User  `json:"user" gorm:"-"` // 非 ORM 关联，由 Service 手动填充
}

func (Comment) TableComment() string {
	return "评论表"
}
