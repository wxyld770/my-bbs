package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const LRUMaxMemoryBytes int64 = 256 * 1024 * 1024

// Stores contains Redis instances with distinct data-retention semantics.
// Persist is for state that must never be evicted under memory pressure, while
// LRU is for disposable cache entries managed by an allkeys-lru Redis instance.
type Stores struct {
	Persist *Client
	LRU     *Client

	closeOnce sync.Once
	closeErr  error
}

// StoresConfig names both Redis roles so callers cannot silently swap two
// positional Config values with different retention semantics.
type StoresConfig struct {
	Persist Config
	LRU     Config
}

// OpenStores opens and checks Persist first, then LRU. It also verifies the
// effective server-side retention policies and that both clients reach
// different Redis processes. This prevents a configuration alias or swapped
// credentials from placing authentication revocations in an evicting cache.
// If any check fails, every connection opened by this function is closed.
func OpenStores(ctx context.Context, cfg StoresConfig) (*Stores, error) {
	persist, err := Open(ctx, cfg.Persist)
	if err != nil {
		return nil, fmt.Errorf("open persist redis: %w", err)
	}

	lru, err := Open(ctx, cfg.LRU)
	if err != nil {
		openErr := fmt.Errorf("open lru redis: %w", err)
		if closeErr := persist.Close(); closeErr != nil {
			return nil, errors.Join(openErr, fmt.Errorf("close persist redis after lru open failure: %w", closeErr))
		}
		return nil, openErr
	}

	stores := &Stores{
		Persist: persist,
		LRU:     lru,
	}
	if err := stores.validateRoles(ctx); err != nil {
		validationErr := fmt.Errorf("validate redis roles: %w", err)
		if closeErr := stores.Close(); closeErr != nil {
			return nil, errors.Join(validationErr, closeErr)
		}
		return nil, validationErr
	}
	return stores, nil
}

type redisInstanceConfig struct {
	runID           string
	maxmemory       int64
	maxmemoryPolicy string
	aofEnabled      bool
}

func (s *Stores) validateRoles(ctx context.Context) error {
	if s == nil || s.Persist == nil || s.LRU == nil {
		return errors.New("both Persist and LRU clients are required")
	}

	persist, err := inspectInstance(ctx, s.Persist)
	if err != nil {
		return fmt.Errorf("inspect persist redis: %w", err)
	}
	if persist.maxmemoryPolicy != "noeviction" {
		return fmt.Errorf("persist redis maxmemory-policy=%q, want noeviction", persist.maxmemoryPolicy)
	}
	if !persist.aofEnabled {
		return errors.New("persist redis AOF is disabled")
	}

	lru, err := inspectInstance(ctx, s.LRU)
	if err != nil {
		return fmt.Errorf("inspect lru redis: %w", err)
	}
	if lru.maxmemory != LRUMaxMemoryBytes {
		return fmt.Errorf("lru redis maxmemory=%d, want %d bytes", lru.maxmemory, LRUMaxMemoryBytes)
	}
	if lru.maxmemoryPolicy != "allkeys-lru" {
		return fmt.Errorf("lru redis maxmemory-policy=%q, want allkeys-lru", lru.maxmemoryPolicy)
	}
	if lru.aofEnabled {
		return errors.New("lru redis AOF must be disabled")
	}
	if persist.runID == lru.runID {
		return errors.New("Persist and LRU resolve to the same Redis process")
	}
	return nil
}

func inspectInstance(ctx context.Context, client *Client) (redisInstanceConfig, error) {
	serverInfo, err := redisInfoValues(ctx, client, "server")
	if err != nil {
		return redisInstanceConfig{}, err
	}
	runID := strings.TrimSpace(serverInfo["run_id"])
	if runID == "" {
		return redisInstanceConfig{}, errors.New("INFO server did not return run_id")
	}

	memoryInfo, err := redisInfoValues(ctx, client, "memory")
	if err != nil {
		return redisInstanceConfig{}, err
	}
	maxmemoryRaw := strings.TrimSpace(memoryInfo["maxmemory"])
	maxmemory, err := strconv.ParseInt(maxmemoryRaw, 10, 64)
	if err != nil || maxmemory < 0 {
		return redisInstanceConfig{}, fmt.Errorf("invalid Redis maxmemory %q", maxmemoryRaw)
	}
	policy := strings.TrimSpace(memoryInfo["maxmemory_policy"])
	if policy == "" {
		return redisInstanceConfig{}, errors.New("INFO memory did not return maxmemory_policy")
	}

	persistenceInfo, err := redisInfoValues(ctx, client, "persistence")
	if err != nil {
		return redisInstanceConfig{}, err
	}
	aofRaw := strings.TrimSpace(persistenceInfo["aof_enabled"])
	if aofRaw != "0" && aofRaw != "1" {
		return redisInstanceConfig{}, fmt.Errorf("invalid Redis aof_enabled %q", aofRaw)
	}
	return redisInstanceConfig{
		runID:           runID,
		maxmemory:       maxmemory,
		maxmemoryPolicy: strings.ToLower(strings.TrimSpace(policy)),
		aofEnabled:      aofRaw == "1",
	}, nil
}

func redisInfoValues(ctx context.Context, client *Client, section string) (map[string]string, error) {
	info, err := client.Info(ctx, section).Result()
	if err != nil {
		return nil, fmt.Errorf("INFO %s: %w", section, err)
	}
	values := make(map[string]string)
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values, nil
}

// Close releases both connection pools. It is safe to call more than once.
func (s *Stores) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		var errs []error
		if s.LRU != nil {
			if err := s.LRU.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close lru redis: %w", err))
			}
		}
		if s.Persist != nil {
			if err := s.Persist.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close persist redis: %w", err))
			}
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}
