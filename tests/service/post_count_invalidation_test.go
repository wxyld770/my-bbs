package service_test

import (
	"context"
	"testing"

	postcache "my-bbs/internal/cache"
	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/tests/testutil"
)

func TestInteractionServices_RefreshOrInvalidatePostCountCache(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	redisClient := testutil.NewTestRedis(t)
	countCache := postcache.NewPostCountCache(redisClient)

	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	user := &model.User{Username: "cache-writer", Password: "x", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	post := &model.Post{UserID: user.ID, Title: "interactions", Content: "body", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	countCache.SetLikeCounts(ctx, map[uint]int64{post.ID: 99})
	likeService := service.NewLikeServiceWithCountCache(likeRepo, postRepo, countCache)
	result, err := likeService.Toggle(ctx, post.ID, user.ID)
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if result.LikeCount != 1 {
		t.Fatalf("Toggle count=%d, want 1", result.LikeCount)
	}
	if got := countCache.GetLikeCounts(ctx, []uint{post.ID})[post.ID]; got != 1 {
		t.Fatalf("cached like count=%d, want exact post-write count 1", got)
	}

	countCache.SetCommentCounts(ctx, map[uint]int64{post.ID: 99})
	commentService := service.NewCommentServiceWithCountCache(commentRepo, postRepo, userRepo, countCache)
	if err := commentService.CreateComment(ctx, post.ID, user.ID, "new comment"); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if got := countCache.GetCommentCounts(ctx, []uint{post.ID}); len(got) != 0 {
		t.Fatalf("comment count cache after create=%v, want invalidated", got)
	}

	comments, err := commentRepo.FindByPostID(ctx, post.ID, 0, 10)
	if err != nil || len(comments) != 1 {
		t.Fatalf("FindByPostID=(%v,%v), want one comment", comments, err)
	}
	countCache.SetCommentCounts(ctx, map[uint]int64{post.ID: 1})
	if err := commentService.DeleteComment(ctx, comments[0].ID, user.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	if got := countCache.GetCommentCounts(ctx, []uint{post.ID}); len(got) != 0 {
		t.Fatalf("comment count cache after delete=%v, want invalidated", got)
	}
}
