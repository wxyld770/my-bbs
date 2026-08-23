package repository_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"
)

func TestSearchReader_UsesCallerContext(t *testing.T) {
	db := testutil.NewTestDB(t)
	reader := gormrepo.NewSearchReader(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := reader.SearchUsers(ctx, "go", 0, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("SearchUsers error=%v, want context.Canceled", err)
	}
	if _, err := reader.SearchPublicPosts(ctx, "go", 0, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("SearchPublicPosts error=%v, want context.Canceled", err)
	}
}
