package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"my-bbs/internal/logger"
	"my-bbs/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PoolConfig 描述 database/sql 连接池参数。
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// InitDB 初始化 GORM 及其底层 database/sql 连接池。
func InitDB(dsn string, pool PoolConfig) (*gorm.DB, *sql.DB, error) {
	// 用标准库 log 的 Writer 构建 GORM Logger，
	// 这样 GORM 的 SQL 日志也走同一个输出（logger.Init 已经接管了 log 的输出）
	gormLog := gormlogger.New(
		log.Default(),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			// SQL 日志只保留语句结构，不记录邀请码、密码摘要等参数值。
			ParameterizedQueries: true,
			Colorful:             false,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         gormLog,
		TranslateError: true,
		// 只保留逻辑关联（Preload），迁移时不创建物理外键
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("获取底层数据库连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	logger.Info(
		"数据库连接成功，连接池 maxOpen=%d maxIdle=%d maxLifetime=%s maxIdleTime=%s",
		pool.MaxOpenConns,
		pool.MaxIdleConns,
		pool.ConnMaxLifetime,
		pool.ConnMaxIdleTime,
	)
	return db, sqlDB, nil
}

// AutoMigrate 自动迁移（独立函数，便于调用）
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Invitation{},
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
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
