package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"

	"gorm.io/gorm"
)

func TestUserService_Login_RejectsMutedUser(t *testing.T) {
	ctx := context.Background()
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	svc := service.NewUserService(repo)

	seedServiceTestUser(t, db, "muted_user", "password1", "Muted")
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
	svc := service.NewUserService(gormrepo.NewUserRepository(db))

	seedServiceTestUser(t, db, "alice", "password1", "Alice")
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
	repo := gormrepo.NewUserRepository(db)
	svc := service.NewUserService(repo)

	seedServiceTestUser(t, db, "bob", "password1", "Bob")
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

func TestUserService_UpdateAvatar_ValidatesAndEnforcesCooldown(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	svc := service.NewUserService(repo)
	user := seedServiceTestUser(t, db, "avatar-user", "password1", "Avatar")

	if err := svc.UpdateAvatar(ctx, user.ID, "  https://cdn.example.com/avatar.webp?size=128  "); err != nil {
		t.Fatalf("first avatar update: %v", err)
	}
	stored, err := repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarUpdatedAt == nil {
		t.Fatalf("load updated user: user=%+v err=%v", stored, err)
	}
	if stored.AvatarURL != "https://cdn.example.com/avatar.webp?size=128" {
		t.Fatalf("avatar_url=%q", stored.AvatarURL)
	}
	firstUpdatedAt := *stored.AvatarUpdatedAt

	// 重试同一链接是幂等成功，且不能刷新冷却时间。
	if err := svc.UpdateAvatar(ctx, user.ID, stored.AvatarURL); err != nil {
		t.Fatalf("same avatar retry: %v", err)
	}
	stored, err = repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarUpdatedAt == nil || !stored.AvatarUpdatedAt.Equal(firstUpdatedAt) {
		t.Fatalf("same URL changed cooldown: user=%+v err=%v", stored, err)
	}

	if err := svc.UpdateAvatar(ctx, user.ID, "https://cdn.example.com/other.png"); !errors.Is(err, bizerr.ErrAvatarUpdateTooFrequent) {
		t.Fatalf("second different avatar error=%v, want cooldown", err)
	}

	oldEnough := time.Now().UTC().Add(-service.AvatarUpdateInterval - time.Minute).Truncate(time.Millisecond)
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("avatar_updated_at", oldEnough).Error; err != nil {
		t.Fatalf("age avatar timestamp: %v", err)
	}
	if err := svc.UpdateAvatar(ctx, user.ID, ""); err != nil {
		t.Fatalf("clear avatar after cooldown: %v", err)
	}
	stored, err = repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarURL != "" || stored.AvatarUpdatedAt == nil {
		t.Fatalf("avatar was not cleared: user=%+v err=%v", stored, err)
	}
}

func TestUserService_UpdateAvatar_RejectsUnsafeURLs(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	svc := service.NewUserService(repo)
	user := seedServiceTestUser(t, db, "avatar-invalid", "password1", "Avatar")

	for _, avatarURL := range []string{
		"http://example.com/avatar.png",
		"/avatar.png",
		"data:image/png;base64,AAAA",
		"javascript:alert(1)",
		"https://user:password@example.com/avatar.png",
		"https:///avatar.png",
		"https://example.com:invalid/avatar.png",
		"https://example.com/" + strings.Repeat("a", 2049),
	} {
		t.Run(avatarURL, func(t *testing.T) {
			if err := svc.UpdateAvatar(ctx, user.ID, avatarURL); !errors.Is(err, bizerr.ErrInvalidAvatarURL) {
				t.Fatalf("UpdateAvatar(%q) error=%v, want invalid URL", avatarURL, err)
			}
		})
	}
}

func TestUserService_Register_SetsNormalStatus(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	invitationRepo := gormrepo.NewInvitationRepository(db)
	svc := service.NewUserServiceWithInvitations(repo, invitationRepo)
	creator := seedServiceTestUser(t, db, "inviter", "password1", "Inviter")
	if err := invitationRepo.CreateInvitation(ctx, &model.Invitation{
		Code:      "NORM01",
		CreatorID: creator.ID,
	}); err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	if err := svc.Register(ctx, "carol", "password1", "Carol", "NORM01"); err != nil {
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

func seedServiceTestUser(
	t *testing.T,
	db *gorm.DB,
	username, password, nickname string,
) *model.User {
	t.Helper()
	hashed, err := bcrypt.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{
		Username: username,
		Password: hashed,
		Nickname: nickname,
		Status:   model.UserStatusNormal,
	}
	if err := db.WithContext(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user %q: %v", username, err)
	}
	return user
}
