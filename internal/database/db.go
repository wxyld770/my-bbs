package database

import (
	"log"
	"time"

	"my-bbs/internal/logger"
	"my-bbs/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func InitDB(dsn string) *gorm.DB {
	// 用标准库 log 的 Writer 构建 GORM Logger，
	// 这样 GORM 的 SQL 日志也走同一个输出（logger.Init 已经接管了 log 的输出）
	gormLog := gormlogger.New(
		log.Default(),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLog,
		// 只保留逻辑关联（Preload），迁移时不创建物理外键
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		logger.Fatal("连接数据库失败: %v", err)
	}

	logger.Info("数据库连接成功")
	return db
}

// AutoMigrate 自动迁移（独立函数，便于调用）
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.PostLike{},
	); err != nil {
		return err
	}
	logger.Info("数据库迁移完成")
	return nil
}

// Close 关闭底层数据库连接（优雅停机时调用）
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
