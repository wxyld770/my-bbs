package database

import (
    "log"
    "my-bbs/internal/model"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func InitDB(dsn string) *gorm.DB {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    
    log.Println("数据库连接成功")
    return db
}

// 自动迁移（独立函数，便于调用）
func AutoMigrate(db *gorm.DB) error {
    if err := db.AutoMigrate(
        &model.User{},
        &model.Post{},
    ); err != nil {
        return err
    }
    log.Println("数据库迁移完成")
    return nil
}