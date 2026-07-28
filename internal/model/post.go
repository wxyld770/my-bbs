package model

// Post 帖子模型
type Post struct {
	BaseModel
	UserID  uint   `gorm:"index;not null" json:"user_id"`
	Title   string `gorm:"type:varchar(255);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`
	Visible string `gorm:"type:tinyint(4);not null;default:1;comment:可见性状态，1所有人可见，0仅自己可见" json:"visible"`
	User    *User  `gorm:"foreignKey:UserID" json:"user"` // 逻辑关联，用于 Preload；物理外键已在 Config 中禁用
}

func (Post) TableComment() string {
	return "帖子信息表"
}
