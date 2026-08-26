package repository_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"

	"gorm.io/gorm"
)

func TestGORMRepositories_DeletesReturnRepositoryNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	postRepo := gormrepo.NewPostRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)

	tests := []struct {
		name   string
		mutate func() error
	}{
		{
			name: "delete post",
			mutate: func() error {
				return postRepo.DeletePost(ctx, 999)
			},
		},
		{
			name: "delete comment",
			mutate: func() error {
				return commentRepo.SoftDelete(ctx, 999)
			},
		},
		{
			name: "delete like",
			mutate: func() error {
				return likeRepo.DeleteByUserAndPost(ctx, 999, 999)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mutate()
			if !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("mutation error = %v, want repository.ErrNotFound", err)
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("mutation leaked gorm.ErrRecordNotFound: %v", err)
			}
		})
	}
}

func TestGORMRepositories_NoOpUpdateIsNotNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)

	user := &model.User{Username: "same-value", Password: "hash", Nickname: "same"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userRepo.UpdateProfile(ctx, user.ID, user.Nickname, user.Introduction); err != nil {
		t.Fatalf("no-op profile update returned error: %v", err)
	}
	if err := userRepo.UpdateUserStatus(ctx, user.ID, user.Status); err != nil {
		t.Fatalf("no-op status update returned error: %v", err)
	}

	post := &model.Post{UserID: user.ID, Title: "title", Content: "content", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := postRepo.UpdatePost(ctx, post); err != nil {
		t.Fatalf("no-op post update returned error: %v", err)
	}
	if err := postRepo.UpdatePostVisible(ctx, post.ID, post.Visible); err != nil {
		t.Fatalf("no-op visibility update returned error: %v", err)
	}
	if err := postRepo.SetPostPinnedUntil(ctx, post.ID, nil); err != nil {
		t.Fatalf("no-op unpin returned error: %v", err)
	}
}

func TestGORMRepositories_UpdatesReturnNotFoundForSoftDeletedRecords(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)

	user := &model.User{Username: "soft-deleted-user", Password: "hash", Nickname: "deleted"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	post := &model.Post{UserID: user.ID, Title: "title", Content: "content", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Delete(post).Error; err != nil {
		t.Fatalf("soft-delete post: %v", err)
	}
	if err := db.Delete(user).Error; err != nil {
		t.Fatalf("soft-delete user: %v", err)
	}

	tests := []struct {
		name   string
		mutate func() error
	}{
		{
			name: "update user status",
			mutate: func() error {
				return userRepo.UpdateUserStatus(ctx, user.ID, model.UserStatusMuted)
			},
		},
		{
			name: "update post",
			mutate: func() error {
				post.Title = "changed"
				return postRepo.UpdatePost(ctx, post)
			},
		},
		{
			name: "update post visibility",
			mutate: func() error {
				return postRepo.UpdatePostVisible(ctx, post.ID, model.VisiblePrivate)
			},
		},
		{
			name: "unpin post",
			mutate: func() error {
				return postRepo.SetPostPinnedUntil(ctx, post.ID, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mutate()
			if !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("mutation error = %v, want repository.ErrNotFound", err)
			}
		})
	}
}

func TestUserRepository_ZeroRowStatusUpdateDoesNotMisreportExistingUserAsNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	userRepo := gormrepo.NewUserRepository(db)

	user := &model.User{Username: "status-race", Password: "hash", Nickname: "status race"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	// RAISE(IGNORE) 模拟 UPDATE 影响 0 行后，回读发现记录仍存在且
	// 状态已与请求值不同的并发场景。
	if err := db.Exec(`
		CREATE TRIGGER ignore_status_update
		BEFORE UPDATE OF status ON users
		BEGIN
			SELECT RAISE(IGNORE);
		END
	`).Error; err != nil {
		t.Fatalf("create status trigger: %v", err)
	}

	if err := userRepo.UpdateUserStatus(ctx, user.ID, model.UserStatusMuted); err != nil {
		t.Fatalf("zero-row update for existing user returned error: %v", err)
	}
	found, err := userRepo.FindUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if found == nil || found.Status != model.UserStatusNormal {
		t.Fatalf("triggered status=%v, want existing normal user", found)
	}
}
