// Package authsession contains application-level Redis semantics for login sessions.
package authsession

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const revokedTokenKeyPrefix = "mybbs:v1:auth:revoked:"

var (
	ErrRedisRequired  = errors.New("token revocation requires redis")
	ErrTokenIDMissing = errors.New("token ID is missing")
)

func revokedTokenKey(tokenID string) string {
	return revokedTokenKeyPrefix + tokenID
}

// IsRevoked reports whether a token ID has been explicitly logged out.
func IsRevoked(ctx context.Context, commands redis.Cmdable, tokenID string) (bool, error) {
	if commands == nil {
		return false, ErrRedisRequired
	}
	if tokenID == "" {
		return false, ErrTokenIDMissing
	}

	count, err := commands.Exists(ctx, revokedTokenKey(tokenID)).Result()
	if err != nil {
		return false, fmt.Errorf("check revoked token: %w", err)
	}
	return count > 0, nil
}

// Revoke marks one token as invalid until its original expiry. An already
// expired token needs no Redis entry and is treated as an idempotent success.
func Revoke(ctx context.Context, commands redis.Cmdable, tokenID string, expiresAt time.Time) error {
	if tokenID == "" {
		return ErrTokenIDMissing
	}

	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return nil
	}
	if commands == nil {
		return ErrRedisRequired
	}
	if err := commands.Set(ctx, revokedTokenKey(tokenID), "1", remaining).Err(); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}
