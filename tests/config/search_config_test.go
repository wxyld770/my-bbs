package config_test

import (
	"strings"
	"testing"

	"my-bbs/internal/config"
)

func TestValidate_RejectsInvalidSearchSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantKey string
	}{
		{name: "timeout", mutate: func(cfg *config.Config) { cfg.SearchTimeout = 0 }, wantKey: "SEARCH_TIMEOUT"},
		{name: "requests", mutate: func(cfg *config.Config) { cfg.RateLimitSearchRequests = 0 }, wantKey: "RATE_LIMIT_SEARCH_REQUESTS"},
		{name: "window", mutate: func(cfg *config.Config) { cfg.RateLimitSearchWindow = 0 }, wantKey: "RATE_LIMIT_SEARCH_WINDOW"},
		{name: "burst", mutate: func(cfg *config.Config) { cfg.RateLimitSearchBurst = 0 }, wantKey: "RATE_LIMIT_SEARCH_BURST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantKey) {
				t.Fatalf("error=%v, want %s validation", err, tt.wantKey)
			}
		})
	}
}
