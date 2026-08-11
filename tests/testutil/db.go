package testutil

import (
	"fmt"
	"strings"
	"testing"

	"my-bbs/internal/database"
	"my-bbs/pkg/jwt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewTestDB 创建独立的内存 SQLite，并完成 AutoMigrate。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
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
