package repository_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"
)

func TestUserRepository_UpdatePasswordHashUsesSessionVersionCAS(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	ctx := context.Background()
	user := &model.User{
		Username: "password-cas-user",
		Password: "original-password-hash",
		Status:   model.UserStatusNormal,
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := repo.UpdatePasswordHash(ctx, user.ID, 0, "winning-password-hash"); err != nil {
		t.Fatalf("first CAS update: %v", err)
	}
	if err := repo.UpdatePasswordHash(ctx, user.ID, 0, "losing-password-hash"); !errors.Is(err, repository.ErrPasswordUpdateConflict) {
		t.Fatalf("stale CAS error=%v, want repository.ErrPasswordUpdateConflict", err)
	}

	stored, err := repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil {
		t.Fatalf("load CAS result: user=%+v err=%v", stored, err)
	}
	if stored.Password != "winning-password-hash" {
		t.Fatalf("stale CAS overwrote winner: password=%q", stored.Password)
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("session_version=%d, want exactly 1", stored.SessionVersion)
	}
}
