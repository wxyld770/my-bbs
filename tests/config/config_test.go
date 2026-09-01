package config_test

import (
	"strings"
	"testing"
	"time"

	"my-bbs/internal/config"
)

func TestValidate_RequiresDBDSNJWTSecretAndRedis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     config.Config
		wantSub string
	}{
		{
			name:    "missing both",
			cfg:     config.Config{},
			wantSub: "DB_DSN",
		},
		{
			name:    "missing jwt",
			cfg:     config.Config{DBDSN: "root:x@tcp(127.0.0.1:3306)/db"},
			wantSub: "JWT_SECRET",
		},
		{
			name:    "missing dsn",
			cfg:     config.Config{JWTSecret: "secret"},
			wantSub: "DB_DSN",
		},
		{
			name:    "missing persist redis",
			cfg:     config.Config{DBDSN: "root:x@tcp(127.0.0.1:3306)/db", JWTSecret: "secret"},
			wantSub: "REDIS_PERSIST_ADDR",
		},
		{
			name: "missing lru redis",
			cfg: config.Config{
				DBDSN:            "root:x@tcp(127.0.0.1:3306)/db",
				JWTSecret:        "secret",
				RedisPersistAddr: "127.0.0.1:6379",
			},
			wantSub: "REDIS_LRU_ADDR",
		},
		{
			name: "redis instances must differ",
			cfg: config.Config{
				DBDSN:            "root:x@tcp(127.0.0.1:3306)/db",
				JWTSecret:        "secret",
				RedisPersistAddr: " 127.0.0.1:6379 ",
				RedisLRUAddr:     "127.0.0.1:6379",
			},
			wantSub: "必须指向不同",
		},
		{
			name: "redis instance host comparison is case insensitive",
			cfg: config.Config{
				DBDSN:            "root:x@tcp(127.0.0.1:3306)/db",
				JWTSecret:        "secret",
				RedisPersistAddr: "LOCALHOST:6379",
				RedisLRUAddr:     "localhost:6379",
			},
			wantSub: "必须指向不同",
		},
		{
			name:    "whitespace only",
			cfg:     config.Config{DBDSN: "  ", JWTSecret: "\t"},
			wantSub: "DB_DSN",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidate_OK(t *testing.T) {
	t.Parallel()
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_RequiresAdminUsernames(t *testing.T) {
	t.Parallel()
	cfg := validConfig()
	cfg.AdminUsernames = "  "

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_USERNAMES") {
		t.Fatalf("error = %v, want ADMIN_USERNAMES validation", err)
	}
}

func TestValidate_RejectsInvalidRateLimitSettings(t *testing.T) {
	t.Parallel()
	cfg := validConfig()
	cfg.RateLimitLoginRequests = 0

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_LOGIN_REQUESTS") {
		t.Fatalf("error = %v, want RATE_LIMIT_LOGIN_REQUESTS validation", err)
	}
}

func TestLoad_ReadsAdminUsernamesFromEnvironment(t *testing.T) {
	t.Setenv("ADMIN_USERNAMES", "admin,moderator")

	cfg := config.Load()
	if cfg.AdminUsernames != "admin,moderator" {
		t.Fatalf("AdminUsernames=%q, want env value", cfg.AdminUsernames)
	}
}

func TestLoad_RedisConfiguration(t *testing.T) {
	t.Run("new persist variables fall back to legacy variables", func(t *testing.T) {
		t.Setenv("REDIS_PERSIST_ADDR", "")
		t.Setenv("REDIS_PERSIST_PASS", "")
		t.Setenv("REDIS_ADDR", "legacy-redis:6379")
		t.Setenv("REDIS_PASS", "legacy-secret")
		t.Setenv("REDIS_LRU_ADDR", "")
		t.Setenv("REDIS_LRU_PASS", "")

		cfg := config.Load()
		if cfg.RedisPersistAddr != "legacy-redis:6379" {
			t.Fatalf("RedisPersistAddr=%q, want legacy-redis:6379", cfg.RedisPersistAddr)
		}
		if cfg.RedisPersistPass != "legacy-secret" {
			t.Fatalf("RedisPersistPass=%q, want legacy-secret", cfg.RedisPersistPass)
		}
		if cfg.RedisLRUAddr != "localhost:6380" {
			t.Fatalf("RedisLRUAddr=%q, want localhost:6380", cfg.RedisLRUAddr)
		}
		if cfg.RedisLRUPass != "" {
			t.Fatalf("RedisLRUPass=%q, want empty default", cfg.RedisLRUPass)
		}
	})

	t.Run("new variables take precedence", func(t *testing.T) {
		t.Setenv("REDIS_PERSIST_ADDR", "persist-redis:6379")
		t.Setenv("REDIS_PERSIST_PASS", "persist-secret")
		t.Setenv("REDIS_ADDR", "legacy-redis:6379")
		t.Setenv("REDIS_PASS", "legacy-secret")
		t.Setenv("REDIS_LRU_ADDR", "lru-redis:6379")
		t.Setenv("REDIS_LRU_PASS", "lru-secret")

		cfg := config.Load()
		if cfg.RedisPersistAddr != "persist-redis:6379" || cfg.RedisPersistPass != "persist-secret" {
			t.Fatalf("persist Redis=(%q, %q), want new values", cfg.RedisPersistAddr, cfg.RedisPersistPass)
		}
		if cfg.RedisLRUAddr != "lru-redis:6379" || cfg.RedisLRUPass != "lru-secret" {
			t.Fatalf("LRU Redis=(%q, %q), want new values", cfg.RedisLRUAddr, cfg.RedisLRUPass)
		}
	})
}

func TestLoad_DBAutoMigrateDefaultsByAppMode(t *testing.T) {
	t.Run("debug defaults enabled", func(t *testing.T) {
		t.Setenv("APP_MODE", "debug")
		t.Setenv("DB_AUTO_MIGRATE", "")

		cfg := config.Load()
		if !cfg.DBAutoMigrate {
			t.Fatal("DBAutoMigrate=false, want true in debug mode")
		}
	})

	t.Run("release defaults disabled", func(t *testing.T) {
		t.Setenv("APP_MODE", "release")
		t.Setenv("DB_AUTO_MIGRATE", "")

		cfg := config.Load()
		if cfg.DBAutoMigrate {
			t.Fatal("DBAutoMigrate=true, want false in release mode")
		}
	})

	t.Run("release mode matching is case insensitive", func(t *testing.T) {
		t.Setenv("APP_MODE", " ReLeAsE ")
		t.Setenv("DB_AUTO_MIGRATE", "")

		cfg := config.Load()
		if cfg.DBAutoMigrate {
			t.Fatal("DBAutoMigrate=true, want false for a release-mode spelling variant")
		}
	})

	t.Run("explicit value overrides release default", func(t *testing.T) {
		t.Setenv("APP_MODE", "release")
		t.Setenv("DB_AUTO_MIGRATE", "true")

		cfg := config.Load()
		if !cfg.DBAutoMigrate {
			t.Fatal("DBAutoMigrate=false, want explicit true")
		}
	})
}

func validConfig() *config.Config {
	return &config.Config{
		DBDSN:                     "root:x@tcp(127.0.0.1:3306)/db",
		DBMaxOpenConns:            25,
		DBMaxIdleConns:            10,
		DBConnMaxLifetime:         30 * time.Minute,
		DBConnMaxIdleTime:         5 * time.Minute,
		RedisPersistAddr:          "127.0.0.1:6379",
		RedisLRUAddr:              "127.0.0.1:6380",
		JWTSecret:                 "a-long-enough-secret",
		AdminUsernames:            "admin",
		AppPort:                   "8080",
		HTTPReadHeaderTimeout:     5 * time.Second,
		HTTPReadTimeout:           10 * time.Second,
		HTTPWriteTimeout:          15 * time.Second,
		HTTPIdleTimeout:           time.Minute,
		HTTPShutdownTimeout:       10 * time.Second,
		HealthCheckTimeout:        2 * time.Second,
		SearchTimeout:             time.Second,
		RateLimitMaxEntries:       20000,
		RateLimitIdleTTL:          15 * time.Minute,
		RateLimitLoginRequests:    10,
		RateLimitLoginWindow:      time.Minute,
		RateLimitLoginBurst:       5,
		RateLimitRegisterRequests: 3,
		RateLimitRegisterWindow:   time.Minute,
		RateLimitRegisterBurst:    2,
		RateLimitSearchRequests:   60,
		RateLimitSearchWindow:     time.Minute,
		RateLimitSearchBurst:      15,
		RateLimitWriteRequests:    30,
		RateLimitWriteWindow:      time.Minute,
		RateLimitWriteBurst:       10,
		RateLimitReadRequests:     600,
		RateLimitReadWindow:       time.Minute,
		RateLimitReadBurst:        120,
	}
}
