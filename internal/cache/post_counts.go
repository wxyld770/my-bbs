// Package cache contains disposable cache-aside adapters. Cached values must
// never be the source of truth and every read path must be able to fall back to
// the database.
package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultPostCountTTL          = 30 * time.Second
	DefaultRedisOperationTimeout = 100 * time.Millisecond

	postCountKeyPrefix = "mybbs:v1:lru:post:"
)

type postCountMetric string

const (
	likeCountMetric    postCountMetric = "likes"
	commentCountMetric postCountMetric = "comments"
)

// PostCountConfig bounds cache staleness and the time a failed cache may add
// to a request. Non-positive values use the package defaults.
type PostCountConfig struct {
	TTL              time.Duration
	OperationTimeout time.Duration
}

// PostCountCache stores only disposable interaction counters. It deliberately
// does not cache post visibility, content, authors, or viewer-specific state.
// A nil Redis client disables the cache and makes every operation a no-op.
type PostCountCache struct {
	redis            redis.Cmdable
	ttl              time.Duration
	operationTimeout time.Duration
}

func NewPostCountCache(commands redis.Cmdable, configs ...PostCountConfig) *PostCountCache {
	config := PostCountConfig{
		TTL:              DefaultPostCountTTL,
		OperationTimeout: DefaultRedisOperationTimeout,
	}
	if len(configs) > 0 {
		if configs[0].TTL > 0 {
			config.TTL = configs[0].TTL
		}
		if configs[0].OperationTimeout > 0 {
			config.OperationTimeout = configs[0].OperationTimeout
		}
	}
	return &PostCountCache{
		redis:            commands,
		ttl:              config.TTL,
		operationTimeout: config.OperationTimeout,
	}
}

func (c *PostCountCache) GetLikeCounts(ctx context.Context, postIDs []uint) map[uint]int64 {
	return c.getMany(ctx, likeCountMetric, postIDs)
}

func (c *PostCountCache) GetCommentCounts(ctx context.Context, postIDs []uint) map[uint]int64 {
	return c.getMany(ctx, commentCountMetric, postIDs)
}

func (c *PostCountCache) SetLikeCounts(ctx context.Context, counts map[uint]int64) {
	c.setMany(ctx, likeCountMetric, counts)
}

func (c *PostCountCache) SetCommentCounts(ctx context.Context, counts map[uint]int64) {
	c.setMany(ctx, commentCountMetric, counts)
}

func (c *PostCountCache) DeleteLikeCounts(ctx context.Context, postIDs ...uint) {
	c.delete(ctx, likeCountMetric, postIDs)
}

func (c *PostCountCache) DeleteCommentCounts(ctx context.Context, postIDs ...uint) {
	c.delete(ctx, commentCountMetric, postIDs)
}

func (c *PostCountCache) getMany(ctx context.Context, metric postCountMetric, postIDs []uint) map[uint]int64 {
	hits := make(map[uint]int64)
	if c == nil || c.redis == nil || len(postIDs) == 0 {
		return hits
	}

	ids := uniqueNonZeroIDs(postIDs)
	if len(ids) == 0 {
		return hits
	}
	keys := make([]string, len(ids))
	for i, postID := range ids {
		keys[i] = postCountKey(metric, postID)
	}

	opCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	values, err := c.redis.MGet(opCtx, keys...).Result()
	if err != nil {
		return hits
	}

	var corruptKeys []string
	for i, raw := range values {
		if raw == nil {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			corruptKeys = append(corruptKeys, keys[i])
			continue
		}
		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil || count < 0 {
			corruptKeys = append(corruptKeys, keys[i])
			continue
		}
		hits[ids[i]] = count
	}
	if len(corruptKeys) > 0 {
		cleanupCtx, cleanupCancel := c.withTimeout(ctx)
		_ = c.redis.Del(cleanupCtx, corruptKeys...).Err()
		cleanupCancel()
	}
	return hits
}

func (c *PostCountCache) setMany(ctx context.Context, metric postCountMetric, counts map[uint]int64) {
	if c == nil || c.redis == nil || len(counts) == 0 {
		return
	}
	opCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, _ = c.redis.Pipelined(opCtx, func(pipe redis.Pipeliner) error {
		for postID, count := range counts {
			if postID == 0 || count < 0 {
				continue
			}
			pipe.Set(opCtx, postCountKey(metric, postID), strconv.FormatInt(count, 10), c.ttl)
		}
		return nil
	})
}

func (c *PostCountCache) delete(ctx context.Context, metric postCountMetric, postIDs []uint) {
	if c == nil || c.redis == nil || len(postIDs) == 0 {
		return
	}
	ids := uniqueNonZeroIDs(postIDs)
	keys := make([]string, len(ids))
	for i, postID := range ids {
		keys[i] = postCountKey(metric, postID)
	}
	if len(keys) == 0 {
		return
	}
	opCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	_ = c.redis.Del(opCtx, keys...).Err()
}

func (c *PostCountCache) withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, c.operationTimeout)
}

func postCountKey(metric postCountMetric, postID uint) string {
	return postCountKeyPrefix + strconv.FormatUint(uint64(postID), 10) + ":" + string(metric) + "-count"
}

func uniqueNonZeroIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
