package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"
)

func TestUserService_GenerateInvitationRequiresAgeOrPublishedPost(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	invitationRepo := gormrepo.NewInvitationRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	svc := service.NewUserServiceWithInvitations(userRepo, invitationRepo)

	youngUser := seedServiceTestUser(t, db, "young-inviter", "password1", "Young")
	createdAt := time.Now().Add(-service.InvitationNewUserPeriod + time.Hour).Truncate(time.Millisecond)
	if err := db.Model(&model.User{}).Where("id = ?", youngUser.ID).
		UpdateColumn("create_time", createdAt).Error; err != nil {
		t.Fatalf("set young user create time: %v", err)
	}

	if _, err := svc.GenerateInvitation(ctx, youngUser.ID); !errors.Is(err, bizerr.ErrInvitationGenerationRestricted) {
		t.Fatalf("young user without post error=%v, want restriction", err)
	}

	// 其他用户的帖子不能解锁当前用户。
	otherUser := seedServiceTestUser(t, db, "other-inviter", "password1", "Other")
	if err := postRepo.CreatePost(ctx, &model.Post{
		UserID:  otherUser.ID,
		Title:   "other post",
		Content: "other content",
		Visible: model.VisiblePublic,
	}); err != nil {
		t.Fatalf("create other user's post: %v", err)
	}
	if _, err := svc.GenerateInvitation(ctx, youngUser.ID); !errors.Is(err, bizerr.ErrInvitationGenerationRestricted) {
		t.Fatalf("other user's post unlocked invitation: %v", err)
	}

	// 私密帖也是一次成功发布，应立即解锁。
	if err := postRepo.CreatePost(ctx, &model.Post{
		UserID:  youngUser.ID,
		Title:   "first post",
		Content: "first content",
		Visible: model.VisiblePrivate,
	}); err != nil {
		t.Fatalf("create young user's post: %v", err)
	}
	if code, err := svc.GenerateInvitation(ctx, youngUser.ID); err != nil || code == "" {
		t.Fatalf("published young user code=%q error=%v", code, err)
	}
}

func TestUserService_GenerateInvitationAllowsSevenDayOldUser(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	invitationRepo := gormrepo.NewInvitationRepository(db)
	svc := service.NewUserServiceWithInvitations(userRepo, invitationRepo)

	user := seedServiceTestUser(t, db, "seven-day-inviter", "password1", "Seven Days")
	createdAt := time.Now().Add(-service.InvitationNewUserPeriod).Truncate(time.Millisecond)
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).
		UpdateColumn("create_time", createdAt).Error; err != nil {
		t.Fatalf("set seven-day create time: %v", err)
	}

	if code, err := svc.GenerateInvitation(ctx, user.ID); err != nil || code == "" {
		t.Fatalf("seven-day user code=%q error=%v", code, err)
	}
}

func TestUserService_GenerateInvitationCountsSoftDeletedPost(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	invitationRepo := gormrepo.NewInvitationRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	svc := service.NewUserServiceWithInvitations(userRepo, invitationRepo)

	user := seedServiceTestUser(t, db, "deleted-post-inviter", "password1", "Deleted Post")
	post := &model.Post{UserID: user.ID, Title: "published", Content: "then deleted", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Delete(post).Error; err != nil {
		t.Fatalf("soft delete post: %v", err)
	}

	if code, err := svc.GenerateInvitation(ctx, user.ID); err != nil || code == "" {
		t.Fatalf("soft-deleted post should retain eligibility: code=%q error=%v", code, err)
	}
}
