package redisstore_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"my-bbs/internal/redisstore"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

var _ redis.UniversalClient = (*redisstore.Client)(nil)

func TestRedisClient_ExposesGoRedisCommands(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := redisstore.Open(context.Background(), redisstore.Config{
		Addr:         server.Addr(),
		PoolSize:     4,
		MinIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	ctx := context.Background()
	if err := client.Set(ctx, "test:string", "value", 0).Err(); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got, err := client.Get(ctx, "test:string").Result(); err != nil || got != "value" {
		t.Fatalf("Get() = (%q, %v), want (value, nil)", got, err)
	}

	if err := client.HSet(ctx, "test:hash", "name", "alice", "age", 18).Err(); err != nil {
		t.Fatalf("HSet() error = %v", err)
	}
	if got, err := client.HGetAll(ctx, "test:hash").Result(); err != nil || got["name"] != "alice" {
		t.Fatalf("HGetAll() = (%v, %v), want name=alice", got, err)
	}

	if _, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.SAdd(ctx, "test:set", "go", "redis")
		pipe.ZAdd(ctx, "test:zset", redis.Z{Score: 10, Member: "post:1"})
		return nil
	}); err != nil {
		t.Fatalf("Pipelined() error = %v", err)
	}

	if err := client.Get(ctx, "test:missing").Err(); !errors.Is(err, redis.Nil) {
		t.Fatalf("Get() missing error = %v, want redis.Nil", err)
	}
}

func TestRedisClient_LifecycleContract(t *testing.T) {
	if _, err := redisstore.Open(context.Background(), redisstore.Config{}); !errors.Is(err, redisstore.ErrInvalidConfig) {
		t.Fatalf("Open() error = %v, want redisstore.ErrInvalidConfig", err)
	}

	server := miniredis.RunT(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if client, err := redisstore.Open(canceledCtx, redisstore.Config{Addr: server.Addr()}); client != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Open() = (%v, %v), want (nil, context.Canceled)", client, err)
	}

	client, err := redisstore.Open(context.Background(), redisstore.Config{Addr: server.Addr()})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := client.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	const closers = 16
	var wg sync.WaitGroup
	errs := make(chan error, closers)
	for range closers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- client.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}

	if err := client.Get(context.Background(), "after-close").Err(); !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("Get() after Close error = %v, want redis.ErrClosed", err)
	}
}
