package config

import (
	"fmt"
	"os"
	"strings"

	"my-bbs/internal/logger"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN     string
	RedisAddr string
	RedisPass string
	JWTSecret string
	AppPort   string
	AppMode   string
	LogDir    string
}

func Load() *Config {
	// 加载 .env
	if err := godotenv.Load("config/.env"); err != nil {
		logger.Warn("未找到 .env 文件，使用系统环境变量")
	}

	return &Config{
		DBDSN:     getEnv("DB_DSN", ""),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASS", ""),
		JWTSecret: getEnv("JWT_SECRET", ""),
		AppPort:   getEnv("APP_PORT", "8080"),
		AppMode:   getEnv("APP_MODE", "debug"),
		LogDir:    getEnv("LOG_DIR", "logs"),
	}
}

// Validate 校验启动必需配置；缺失时返回错误，由调用方直接退出。
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("配置未加载")
	}
	if strings.TrimSpace(c.DBDSN) == "" {
		return fmt.Errorf("缺少必要配置 DB_DSN")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("缺少必要配置 JWT_SECRET")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
