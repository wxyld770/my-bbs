package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"my-bbs/internal/authsession"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestUserServiceLogoutRevokesOnlyForRemainingTokenLifetime(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	userService := service.NewUserServiceWithRedis(nil, client)
	expiresAt := time.Now().Add(5 * time.Minute)

	if err := userService.Logout(context.Background(), "logout-token-id", expiresAt); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	revoked, err := authsession.IsRevoked(context.Background(), client, "logout-token-id")
	if err != nil || !revoked {
		t.Fatalf("IsRevoked() = (%t, %v), want (true, nil)", revoked, err)
	}
	server.FastForward(6 * time.Minute)
	revoked, err = authsession.IsRevoked(context.Background(), client, "logout-token-id")
	if err != nil || revoked {
		t.Fatalf("IsRevoked() after expiry = (%t, %v), want (false, nil)", revoked, err)
	}

	if err := userService.Logout(context.Background(), "already-expired", time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("Logout(expired) error = %v", err)
	}
	revoked, err = authsession.IsRevoked(context.Background(), client, "already-expired")
	if err != nil || revoked {
		t.Fatalf("expired Logout IsRevoked() = (%t, %v), want (false, nil)", revoked, err)
	}
}

func TestUserServiceLogoutRejectsMissingJTIAndMapsRedisFailure(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	userService := service.NewUserServiceWithRedis(nil, client)

	if err := userService.Logout(context.Background(), "", time.Now().Add(time.Minute)); !errors.Is(err, bizerr.ErrInvalidToken) {
		t.Fatalf("Logout(missing JTI) error = %v, want ErrInvalidToken", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := userService.Logout(context.Background(), "redis-failure", time.Now().Add(time.Minute)); !errors.Is(err, bizerr.ErrServiceUnavailable) {
		t.Fatalf("Logout(redis failure) error = %v, want ErrServiceUnavailable", err)
	}
}
