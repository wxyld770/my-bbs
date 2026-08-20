package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"my-bbs/internal/logger"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN                 string
	DBMaxOpenConns        int
	DBMaxIdleConns        int
	DBConnMaxLifetime     time.Duration
	DBConnMaxIdleTime     time.Duration
	RedisAddr             string
	RedisPass             string
	JWTSecret             string
	AppPort               string
	AppMode               string
	LogDir                string
	HTTPReadHeaderTimeout time.Duration
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	HTTPShutdownTimeout   time.Duration
	HealthCheckTimeout    time.Duration
}

func Load() *Config {
	// 加载 .env
	if err := godotenv.Load("config/.env"); err != nil {
		logger.Warn("未找到 .env 文件，使用系统环境变量")
	}

	return &Config{
		DBDSN:                 getEnv("DB_DSN", ""),
		DBMaxOpenConns:        getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:        getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:     getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime:     getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		RedisAddr:             getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:             getEnv("REDIS_PASS", ""),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		AppPort:               getEnv("APP_PORT", "8080"),
		AppMode:               getEnv("APP_MODE", "debug"),
		LogDir:                getEnv("LOG_DIR", "logs"),
		HTTPReadHeaderTimeout: getEnvDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		HTTPReadTimeout:       getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:      getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		HTTPIdleTimeout:       getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		HTTPShutdownTimeout:   getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		HealthCheckTimeout:    getEnvDuration("HEALTH_CHECK_TIMEOUT", 2*time.Second),
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
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("缺少必要配置 REDIS_ADDR")
	}
	port, err := strconv.Atoi(c.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT 必须是 1-65535 之间的整数")
	}
	if c.DBMaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS 必须大于 0")
	}
	if c.DBMaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS 不能小于 0")
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS 不能大于 DB_MAX_OPEN_CONNS")
	}
	positiveDurations := []struct {
		name  string
		value time.Duration
	}{
		{"DB_CONN_MAX_LIFETIME", c.DBConnMaxLifetime},
		{"DB_CONN_MAX_IDLE_TIME", c.DBConnMaxIdleTime},
		{"HTTP_READ_HEADER_TIMEOUT", c.HTTPReadHeaderTimeout},
		{"HTTP_READ_TIMEOUT", c.HTTPReadTimeout},
		{"HTTP_WRITE_TIMEOUT", c.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", c.HTTPIdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", c.HTTPShutdownTimeout},
		{"HEALTH_CHECK_TIMEOUT", c.HealthCheckTimeout},
	}
	for _, item := range positiveDurations {
		if item.value <= 0 {
			return fmt.Errorf("%s 必须大于 0", item.name)
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		logger.Warn("配置 %s=%q 不是有效整数，使用默认值 %d", key, raw, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		logger.Warn("配置 %s=%q 不是有效时长，使用默认值 %s", key, raw, defaultValue)
		return defaultValue
	}
	return value
}
