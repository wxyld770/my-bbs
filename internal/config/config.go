package config

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    DBDSN      string
    RedisAddr  string
    RedisPass  string
    JWTSecret  string
    AppPort    string
}

func Load() *Config {
    // 加载 .env
    if err := godotenv.Load(); err != nil {
        log.Println("未找到 .env 文件，使用系统环境变量")
    }

    return &Config{
        DBDSN:     getEnv("DB_DSN", ""),
        RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPass: getEnv("REDIS_PASS", ""),
        JWTSecret: getEnv("JWT_SECRET", "default-secret-key"),
        AppPort:   getEnv("APP_PORT", "8080"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}