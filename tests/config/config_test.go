package config_test

import (
	"strings"
	"testing"

	"my-bbs/internal/config"
)

func TestValidate_RequiresDBDSNAndJWTSecret(t *testing.T) {
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
	cfg := &config.Config{
		DBDSN:     "root:x@tcp(127.0.0.1:3306)/db",
		JWTSecret: "a-long-enough-secret",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
