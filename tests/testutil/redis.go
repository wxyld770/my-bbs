package testutil

import (
	"context"
	"testing"

	"my-bbs/internal/redisstore"

	"github.com/alicebob/miniredis/v2"
)

// NewTestRedis starts an isolated Redis-compatible server and returns the
// production lifecycle client. Cleanup is registered on t.
func NewTestRedis(t *testing.T) *redisstore.Client {
	t.Helper()
	server := miniredis.RunT(t)
	client, err := redisstore.Open(context.Background(), redisstore.Config{Addr: server.Addr()})
	if err != nil {
		t.Fatalf("open test Redis: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close test Redis: %v", err)
		}
	})
	return client
}
