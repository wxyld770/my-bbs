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
	if len(result.List) != 1 || result.List[0].Post.Title != "public" {
		t.Fatalf("expected only public post, got %+v", result.List)
	}

	_, err = svc.GetPublicPostsByUser(ctx, 99999, pagination.Query{PageNo: 1, PageSize: 10})
	if !errors.Is(err, bizerr.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestPostService_GetAllPosts_IncludesInteractionCounts(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)
	svc := service.NewPostService(postRepo, userRepo, commentRepo, likeRepo)

	author := &model.User{Username: "countauthor", Password: "x", Nickname: "作者", Status: model.UserStatusNormal}
	reader := &model.User{Username: "countreader", Password: "x", Nickname: "读者", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	if err := userRepo.CreateUser(ctx, reader); err != nil {
		t.Fatalf("create reader: %v", err)
	}

	activePost := &model.Post{UserID: author.ID, Title: "有互动", Content: "正文", Visible: model.VisiblePublic}
	quietPost := &model.Post{UserID: author.ID, Title: "已撤销互动", Content: "正文", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, activePost); err != nil {
		t.Fatalf("create active post: %v", err)
	}
	if err := postRepo.CreatePost(ctx, quietPost); err != nil {
		t.Fatalf("create quiet post: %v", err)
	}

	for _, like := range []*model.PostLike{
		{PostID: activePost.ID, UserID: author.ID},
		{PostID: activePost.ID, UserID: reader.ID},
		{PostID: quietPost.ID, UserID: reader.ID},
	} {
		if err := likeRepo.Create(ctx, like); err != nil {
			t.Fatalf("create like: %v", err)
		}
	}
	if err := likeRepo.DeleteByUserAndPost(ctx, reader.ID, quietPost.ID); err != nil {
		t.Fatalf("remove like: %v", err)
	}

	for _, comment := range []*model.Comment{
		{PostID: activePost.ID, UserID: author.ID, Content: "第一条"},
		{PostID: activePost.ID, UserID: reader.ID, Content: "第二条"},
	} {
		if err := commentRepo.Create(ctx, comment); err != nil {
			t.Fatalf("create comment: %v", err)
		}
	}
	deletedComment := &model.Comment{PostID: quietPost.ID, UserID: reader.ID, Content: "随后删除"}
	if err := commentRepo.Create(ctx, deletedComment); err != nil {
		t.Fatalf("create deleted comment: %v", err)
	}
	if err := commentRepo.SoftDelete(ctx, deletedComment.ID); err != nil {
		t.Fatalf("delete comment: %v", err)
	}

	result, err := svc.GetAllPosts(ctx, pagination.Query{PageNo: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetAllPosts: %v", err)
	}
	if len(result.List) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(result.List))
	}

	counts := make(map[uint]service.PostSummary, len(result.List))
	for _, item := range result.List {
		counts[item.Post.ID] = item
	}
	if got := counts[activePost.ID]; got.LikeCount != 2 || got.CommentCount != 2 {
		t.Fatalf("active post counts = likes %d, comments %d; want 2, 2", got.LikeCount, got.CommentCount)
	}
	if got := counts[quietPost.ID]; got.LikeCount != 0 || got.CommentCount != 0 {
		t.Fatalf("quiet post counts = likes %d, comments %d; want 0, 0", got.LikeCount, got.CommentCount)
	}
}
