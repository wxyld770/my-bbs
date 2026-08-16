package service_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/tests/testutil"
)

func TestPostService_GetPostByID_AuthorCanReadPrivate(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	svc := service.NewPostService(
		postRepo,
		userRepo,
		gormrepo.NewCommentRepository(db),
		gormrepo.NewLikeRepository(db),
	)

	author := &model.User{Username: "author1", Password: "x", Nickname: "A", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	other := &model.User{Username: "other1", Password: "x", Nickname: "O", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, other); err != nil {
		t.Fatalf("create other: %v", err)
	}

	post := &model.Post{
		UserID:  author.ID,
		Title:   "secret",
		Content: "private content",
		Visible: model.VisiblePrivate,
	}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	detail, err := svc.GetPostByID(ctx, post.ID, author.ID)
	if err != nil {
		t.Fatalf("author should read private post: %v", err)
	}
	if detail.Post.Title != "secret" {
		t.Fatalf("unexpected title: %s", detail.Post.Title)
	}

	_, err = svc.GetPostByID(ctx, post.ID, other.ID)
	if !errors.Is(err, bizerr.ErrPostNotFound) {
		t.Fatalf("other should not read private, got %v", err)
	}

	_, err = svc.GetPostByID(ctx, post.ID, 0)
	if !errors.Is(err, bizerr.ErrPostNotFound) {
		t.Fatalf("anonymous should not read private, got %v", err)
	}
}

func TestPostService_GetPublicPostsByUser_FiltersPrivate(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	svc := service.NewPostService(
		postRepo,
		userRepo,
		gormrepo.NewCommentRepository(db),
		gormrepo.NewLikeRepository(db),
	)

	user := &model.User{Username: "pubuser", Password: "x", Nickname: "P", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := postRepo.CreatePost(ctx, &model.Post{
		UserID: user.ID, Title: "public", Content: "p", Visible: model.VisiblePublic,
	}); err != nil {
		t.Fatalf("create public: %v", err)
	}
	if err := postRepo.CreatePost(ctx, &model.Post{
		UserID: user.ID, Title: "private", Content: "s", Visible: model.VisiblePrivate,
	}); err != nil {
		t.Fatalf("create private: %v", err)
	}

	result, err := svc.GetPublicPostsByUser(ctx, user.ID, pagination.Query{PageNo: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetPublicPostsByUser: %v", err)
	}
	if len(result.List) != 1 || result.List[0].Title != "public" {
		t.Fatalf("expected only public post, got %+v", result.List)
	}

	_, err = svc.GetPublicPostsByUser(ctx, 99999, pagination.Query{PageNo: 1, PageSize: 10})
	if !errors.Is(err, bizerr.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}
