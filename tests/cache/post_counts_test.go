package cache_test

import (
	"context"
	"testing"
	"time"

	postcache "my-bbs/internal/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPostCountCache_SetGetExpireAndSeparateMetrics(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	counts := postcache.NewPostCountCache(client, postcache.PostCountConfig{
		TTL:              30 * time.Second,
		OperationTimeout: time.Second,
	})
	ctx := context.Background()

	counts.SetLikeCounts(ctx, map[uint]int64{1: 0, 2: 7})
	counts.SetCommentCounts(ctx, map[uint]int64{1: 3})

	if got := counts.GetLikeCounts(ctx, []uint{1, 2, 3, 2, 0}); !equalCounts(got, map[uint]int64{1: 0, 2: 7}) {
		t.Fatalf("like counts=%v, want cached zero and non-zero values", got)
	}
	if got := counts.GetCommentCounts(ctx, []uint{1, 2}); !equalCounts(got, map[uint]int64{1: 3}) {
		t.Fatalf("comment counts=%v, want only post 1", got)
	}

	server.FastForward(31 * time.Second)
	if got := counts.GetLikeCounts(ctx, []uint{1, 2}); len(got) != 0 {
		t.Fatalf("expired like counts=%v, want cache miss", got)
	}
	if got := counts.GetCommentCounts(ctx, []uint{1}); len(got) != 0 {
		t.Fatalf("expired comment counts=%v, want cache miss", got)
	}
}

func TestPostCountCache_CorruptValuesAndRedisFailureAreCacheMisses(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	counts := postcache.NewPostCountCache(client, postcache.PostCountConfig{OperationTimeout: 20 * time.Millisecond})
	ctx := context.Background()

	server.Set("mybbs:v1:lru:post:1:likes-count", "not-a-number")
	server.Set("mybbs:v1:lru:post:2:likes-count", "-1")
	if got := counts.GetLikeCounts(ctx, []uint{1, 2}); len(got) != 0 {
		t.Fatalf("corrupt values=%v, want cache misses", got)
	}
	if server.Exists("mybbs:v1:lru:post:1:likes-count") || server.Exists("mybbs:v1:lru:post:2:likes-count") {
		t.Fatal("corrupt cache values were not removed")
	}

	server.Close()
	if got := counts.GetLikeCounts(ctx, []uint{1}); len(got) != 0 {
		t.Fatalf("unavailable Redis counts=%v, want cache miss", got)
	}
	counts.SetLikeCounts(ctx, map[uint]int64{1: 4})
	counts.DeleteLikeCounts(ctx, 1)
	_ = client.Close()
}

func TestPostCountCache_NilClientIsDisabled(t *testing.T) {
	counts := postcache.NewPostCountCache(nil)
	ctx := context.Background()
	counts.SetLikeCounts(ctx, map[uint]int64{1: 1})
	counts.SetCommentCounts(ctx, map[uint]int64{1: 1})
	counts.DeleteLikeCounts(ctx, 1)
	counts.DeleteCommentCounts(ctx, 1)
	if got := counts.GetLikeCounts(ctx, []uint{1}); len(got) != 0 {
		t.Fatalf("disabled cache returned %v", got)
	}
}

func equalCounts(left, right map[uint]int64) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range right {
		if left[key] != value {
			return false
		}
	}
	return true
}
