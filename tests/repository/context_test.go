package repository_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"
)

func TestUserRepository_HonorsCanceledContext(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.FindUserByID(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
