package model

import (
    "gorm.io/gorm"
)

// User 用户模型
type User struct {
    gorm.Model          // 自带 ID、CreatedAt、UpdatedAt、DeletedAt（软删除）
    Username   string   `gorm:"type:varchar(64);not null;comment:用户名" json:"username"`
    Password   string   `gorm:"type:varchar(255);not null" json:"-"` // 不返回给前端
    Nickname   string   `gorm:"type:varchar(64);comment:用户昵称" json:"nickname"`
	Status     uint     `gorm:"type:tinyint(4);default:1;comment:用户状态，1:正常，0禁言" json:"status"`
	Introduction   string   `gorm:"type:varchar(1024);comment:个人介绍" json:"introduction"`
}

// 实现 TableComment 方法，返回表注释
func (User) TableComment() string {
    return "用户信息表"
}