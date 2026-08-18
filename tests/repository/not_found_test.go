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

	post := &model.Post{UserID: user.ID, Title: "title", Content: "content", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := postRepo.UpdatePostVisible(ctx, post.ID, post.Visible); err != nil {
		t.Fatalf("no-op visibility update returned error: %v", err)
	}
}
