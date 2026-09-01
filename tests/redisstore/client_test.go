package redisstore_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"my-bbs/internal/redisstore"

	"github.com/alicebob/miniredis/v2"
	miniredisserver "github.com/alicebob/miniredis/v2/server"
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

func TestOpenStores_OpensNamedIndependentInstances(t *testing.T) {
	persistServer := miniredis.RunT(t)
	lruServer := miniredis.RunT(t)
	configureRedisRole(persistServer, "persist-run", 0, "noeviction", true)
	configureRedisRole(lruServer, "lru-run", redisstore.LRUMaxMemoryBytes, "allkeys-lru", false)

	stores, err := redisstore.OpenStores(
		context.Background(),
		redisstore.StoresConfig{
			Persist: redisstore.Config{Addr: persistServer.Addr()},
			LRU:     redisstore.Config{Addr: lruServer.Addr()},
		},
	)
	if err != nil {
		t.Fatalf("OpenStores() error = %v", err)
	}
	if stores.Persist == nil || stores.LRU == nil {
		t.Fatalf("OpenStores() = %#v, want both named clients", stores)
	}

	ctx := context.Background()
	if err := stores.Persist.Set(ctx, "persist:key", "persist", 0).Err(); err != nil {
		t.Fatalf("Persist.Set() error = %v", err)
	}
	if err := stores.LRU.Set(ctx, "lru:key", "lru", 0).Err(); err != nil {
		t.Fatalf("LRU.Set() error = %v", err)
	}
	if !persistServer.Exists("persist:key") || persistServer.Exists("lru:key") {
		t.Fatal("Persist client did not stay isolated on the persist Redis instance")
	}
	if !lruServer.Exists("lru:key") || lruServer.Exists("persist:key") {
		t.Fatal("LRU client did not stay isolated on the LRU Redis instance")
	}

	if err := stores.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := stores.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	waitForNoConnections(t, "persist", persistServer)
	waitForNoConnections(t, "LRU", lruServer)
}

func TestOpenStores_RejectsUnsafeRoleConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		persistRunID      string
		persistPolicy     string
		persistAOF        bool
		lruRunID          string
		lruMaxmemory      int64
		lruPolicy         string
		lruAOF            bool
		wantErrorContains string
	}{
		{
			name:              "same physical Redis identity",
			persistRunID:      "same-run",
			persistPolicy:     "noeviction",
			persistAOF:        true,
			lruRunID:          "same-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes,
			lruPolicy:         "allkeys-lru",
			wantErrorContains: "same Redis process",
		},
		{
			name:              "persist can evict",
			persistRunID:      "persist-run",
			persistPolicy:     "allkeys-lru",
			persistAOF:        true,
			lruRunID:          "lru-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes,
			lruPolicy:         "allkeys-lru",
			wantErrorContains: "want noeviction",
		},
		{
			name:              "lru memory differs",
			persistRunID:      "persist-run",
			persistPolicy:     "noeviction",
			persistAOF:        true,
			lruRunID:          "lru-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes / 2,
			lruPolicy:         "allkeys-lru",
			wantErrorContains: "want 268435456 bytes",
		},
		{
			name:              "lru policy differs",
			persistRunID:      "persist-run",
			persistPolicy:     "noeviction",
			persistAOF:        true,
			lruRunID:          "lru-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes,
			lruPolicy:         "noeviction",
			wantErrorContains: "want allkeys-lru",
		},
		{
			name:              "persist AOF disabled",
			persistRunID:      "persist-run",
			persistPolicy:     "noeviction",
			persistAOF:        false,
			lruRunID:          "lru-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes,
			lruPolicy:         "allkeys-lru",
			wantErrorContains: "AOF is disabled",
		},
		{
			name:              "lru AOF enabled",
			persistRunID:      "persist-run",
			persistPolicy:     "noeviction",
			persistAOF:        true,
			lruRunID:          "lru-run",
			lruMaxmemory:      redisstore.LRUMaxMemoryBytes,
			lruPolicy:         "allkeys-lru",
			lruAOF:            true,
			wantErrorContains: "AOF must be disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persistServer := miniredis.RunT(t)
			lruServer := miniredis.RunT(t)
			configureRedisRole(persistServer, tt.persistRunID, 0, tt.persistPolicy, tt.persistAOF)
			configureRedisRole(lruServer, tt.lruRunID, tt.lruMaxmemory, tt.lruPolicy, tt.lruAOF)

			stores, err := redisstore.OpenStores(context.Background(), redisstore.StoresConfig{
				Persist: redisstore.Config{Addr: persistServer.Addr()},
				LRU:     redisstore.Config{Addr: lruServer.Addr()},
			})
			if stores != nil {
				t.Fatalf("OpenStores() stores=%#v, want nil", stores)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Fatalf("OpenStores() error=%v, want substring %q", err, tt.wantErrorContains)
			}
			waitForNoConnections(t, "persist after role rejection", persistServer)
			waitForNoConnections(t, "LRU after role rejection", lruServer)
		})
	}
}

func TestOpenStores_ClosesPersistWhenLRUOpenFails(t *testing.T) {
	persistServer := miniredis.RunT(t)

	stores, err := redisstore.OpenStores(
		context.Background(),
		redisstore.StoresConfig{
			Persist: redisstore.Config{Addr: persistServer.Addr()},
		},
	)
	if stores != nil {
		t.Fatalf("OpenStores() stores = %#v, want nil", stores)
	}
	if err == nil || !strings.Contains(err.Error(), "open lru redis") || !errors.Is(err, redisstore.ErrInvalidConfig) {
		t.Fatalf("OpenStores() error = %v, want named LRU ErrInvalidConfig", err)
	}
	if got := persistServer.TotalConnectionCount(); got == 0 {
		t.Fatal("persist Redis was never opened before the LRU failure")
	}
	waitForNoConnections(t, "persist after LRU failure", persistServer)
}

func waitForNoConnections(t *testing.T, name string, server *miniredis.Miniredis) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for server.CurrentConnectionCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := server.CurrentConnectionCount(); got != 0 {
		t.Fatalf("%s connection count = %d, want 0", name, got)
	}
}

func configureRedisRole(server *miniredis.Miniredis, runID string, maxmemory int64, policy string, aof bool) {
	server.Server().SetPreHook(func(peer *miniredisserver.Peer, command string, args ...string) bool {
		if !strings.EqualFold(command, "INFO") || len(args) != 1 {
			return false
		}
		var response string
		switch strings.ToLower(args[0]) {
		case "server":
			response = "# Server\r\nrun_id:" + runID + "\r\n"
		case "memory":
			response = "# Memory\r\nmaxmemory:" + strconv.FormatInt(maxmemory, 10) + "\r\nmaxmemory_policy:" + policy + "\r\n"
		case "persistence":
			aofValue := "0"
			if aof {
				aofValue = "1"
			}
			response = "# Persistence\r\naof_enabled:" + aofValue + "\r\n"
		default:
			return false
		}
		peer.WriteBulk(response)
		return true
	})
}
