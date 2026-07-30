package model

import "time"

// PostLike 帖子点赞（硬删除取消赞，配合唯一索引防重复）
type PostLike struct {
	ID         uint      `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"update_time"`
	PostID     uint      `gorm:"uniqueIndex:uk_like_user_post;not null" json:"post_id"`
	UserID     uint      `gorm:"uniqueIndex:uk_like_user_post;not null" json:"user_id"`
}

func (PostLike) TableName() string {
	return "post_likes"
}

func (PostLike) TableComment() string {
	return "帖子点赞表"
}
