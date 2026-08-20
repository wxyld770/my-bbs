// Package redisstore manages Redis configuration and connection lifecycle.
package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrInvalidConfig indicates that Redis cannot be opened with the supplied configuration.
var ErrInvalidConfig = errors.New("invalid redis config")

// Config contains standalone Redis connection and pool settings.
// Zero values for pool sizes, retries, and timeouts use go-redis defaults.
type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

// universalClient keeps the embedded go-redis implementation private while
// promoting its exported command methods to Client.
type universalClient interface {
	redis.UniversalClient
}

// Client exposes the complete go-redis API and owns the connection pool
// lifecycle. Application services should receive it as redis.Cmdable so their
// dependency does not include Close.
type Client struct {
	universalClient
	closeOnce sync.Once
	closeErr  error
}

var _ redis.UniversalClient = (*Client)(nil)

// Open validates the configuration, creates the connection pool, and verifies
// connectivity with PING. The caller controls the startup timeout through ctx.
func Open(ctx context.Context, cfg Config) (*Client, error) {
	options, err := redisOptions(cfg)
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(options)
	client := &Client{universalClient: redisClient}
	if err := client.Check(ctx); err != nil {
		_ = redisClient.Close()
		return nil, err
	}
	return client, nil
}

// Check verifies that Redis is reachable using the caller's context.
func (c *Client) Check(ctx context.Context) error {
	if c == nil || c.universalClient == nil {
		return redis.ErrClosed
	}
	if err := c.universalClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// PingContext adapts Client to the application's readiness-check contract.
func (c *Client) PingContext(ctx context.Context) error {
	return c.Check(ctx)
}

// Close releases the Redis connection pool. It is safe to call more than once.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		c.closeErr = c.universalClient.Close()
	})
	return c.closeErr
}

func redisOptions(cfg Config) (*redis.Options, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, fmt.Errorf("%w: Addr is required", ErrInvalidConfig)
	}
	if cfg.DB < 0 {
		return nil, fmt.Errorf("%w: DB cannot be negative", ErrInvalidConfig)
	}
	if cfg.PoolSize < 0 || cfg.MinIdleConns < 0 {
		return nil, fmt.Errorf("%w: pool sizes cannot be negative", ErrInvalidConfig)
	}
	if cfg.PoolSize > 0 && cfg.MinIdleConns > cfg.PoolSize {
		return nil, fmt.Errorf("%w: MinIdleConns cannot exceed PoolSize", ErrInvalidConfig)
	}
	if cfg.MaxRetries < -1 {
		return nil, fmt.Errorf("%w: MaxRetries cannot be less than -1", ErrInvalidConfig)
	}
	if cfg.DialTimeout < 0 || cfg.ReadTimeout < 0 || cfg.WriteTimeout < 0 || cfg.PoolTimeout < 0 {
		return nil, fmt.Errorf("%w: timeouts cannot be negative", ErrInvalidConfig)
	}

	return &redis.Options{
		Addr:                  addr,
		Username:              cfg.Username,
		Password:              cfg.Password,
		DB:                    cfg.DB,
		PoolSize:              cfg.PoolSize,
		MinIdleConns:          cfg.MinIdleConns,
		MaxRetries:            cfg.MaxRetries,
		DialTimeout:           cfg.DialTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		PoolTimeout:           cfg.PoolTimeout,
		ContextTimeoutEnabled: true,
	}, nil
}
