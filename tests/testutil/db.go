package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"

	"my-bbs/internal/database"
	"my-bbs/pkg/jwt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBSequence atomic.Uint64

// NewTestDB 创建独立的内存 SQLite，并完成 AutoMigrate。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:my-bbs-test-%d?mode=memory&cache=shared", testDBSequence.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		TranslateError:                           true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite connection pool: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite connection pool: %v", err)
		}
	})
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// InitJWT 为测试初始化 JWT 密钥。
func InitJWT(t *testing.T) {
	t.Helper()
	jwt.Init("test-jwt-secret-for-unit-tests")
}
