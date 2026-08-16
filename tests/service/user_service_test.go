package service_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"
)

func TestUserService_Login_RejectsMutedUser(t *testing.T) {
	ctx := context.Background()
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)

	if err := svc.Register(ctx, "muted_user", "password1", "Muted"); err != nil {
		t.Fatalf("register: %v", err)
	}
	user, err := repo.FindUserByUsername(ctx, "muted_user")
	if err != nil || user == nil {
		t.Fatalf("find user: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("status", model.UserStatusMuted).Error; err != nil {
		t.Fatalf("mute user: %v", err)
	}

	_, err = svc.Login(ctx, "muted_user", "password1")
	if !errors.Is(err, bizerr.ErrUserMuted) {
		t.Fatalf("want ErrUserMuted, got %v", err)
	}
}

func TestUserService_Login_OK(t *testing.T) {
	ctx := context.Background()
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(db))

	if err := svc.Register(ctx, "alice", "password1", "Alice"); err != nil {
		t.Fatalf("register: %v", err)
	}
	token, err := svc.Login(ctx, "alice", "password1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestUserService_GetPublicProfile(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)

	if err := svc.Register(ctx, "bob", "password1", "Bob"); err != nil {
		t.Fatalf("register: %v", err)
	}
	user, err := repo.FindUserByUsername(ctx, "bob")
	if err != nil || user == nil {
		t.Fatalf("find: %v", err)
	}

	profile, err := svc.GetPublicProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPublicProfile: %v", err)
	}
	if profile.Username != "bob" || profile.Nickname != "Bob" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	_, err = svc.GetPublicProfile(ctx, 99999)
	if !errors.Is(err, bizerr.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Register_SetsNormalStatus(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)

	if err := svc.Register(ctx, "carol", "password1", "Carol"); err != nil {
		t.Fatalf("register: %v", err)
	}
	user, err := repo.FindUserByUsername(ctx, "carol")
	if err != nil || user == nil {
		t.Fatalf("find: %v", err)
	}
	if user.Status != model.UserStatusNormal {
		t.Fatalf("status = %d, want %d", user.Status, model.UserStatusNormal)
	}
}
