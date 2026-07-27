package config

import (
	"os"

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
		JWTSecret: getEnv("JWT_SECRET", "default-secret-key"),
		AppPort:   getEnv("APP_PORT", "8080"),
		AppMode:   getEnv("APP_MODE", "debug"),
		LogDir:    getEnv("LOG_DIR", "logs"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
