package service_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	postcache "my-bbs/internal/cache"
	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/tests/testutil"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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
	if detail.Post.Title != "secret" || detail.Post.Content != "private content" {
		t.Fatalf("detail must preserve title and content: %+v", detail.Post)
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
	if result.List[0].Post.Content != "" {
		t.Fatalf("public list query must not load content: %+v", result.List[0].Post)
	}
	myPosts, err := svc.GetPostsByUser(ctx, user.ID, pagination.Query{PageNo: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetPostsByUser: %v", err)
	}
	if len(myPosts.List) != 2 {
		t.Fatalf("expected public and private posts, got %+v", myPosts.List)
	}
	for _, item := range myPosts.List {
		if item.Post.Content != "" {
			t.Fatalf("my-posts list query must not load content: %+v", item.Post)
		}
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
	for _, item := range result.List {
		if item.Post.Content != "" {
			t.Fatalf("list query must not load content: %+v", item.Post)
		}
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

func TestPostService_PostCountsUseLRUAndFallBackToDatabase(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	userRepo := gormrepo.NewUserRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)

	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	countCache := postcache.NewPostCountCache(redisClient, postcache.PostCountConfig{
		TTL:              30 * time.Second,
		OperationTimeout: 20 * time.Millisecond,
	})
	svc := service.NewPostServiceWithCountCache(postRepo, userRepo, commentRepo, likeRepo, countCache)

	author := &model.User{Username: "cache-author", Password: "x", Status: model.UserStatusNormal}
	if err := userRepo.CreateUser(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	post := &model.Post{UserID: author.ID, Title: "cached counts", Content: "body", Visible: model.VisiblePublic}
	if err := postRepo.CreatePost(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	like := &model.PostLike{PostID: post.ID, UserID: author.ID}
	if err := likeRepo.Create(ctx, like); err != nil {
		t.Fatalf("create like: %v", err)
	}
	comment := &model.Comment{PostID: post.ID, UserID: author.ID, Content: "cached"}
	if err := commentRepo.Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	assertPostCounts(t, svc, post.ID, 1, 1)
	if !redisServer.Exists("mybbs:v1:lru:post:"+strconv.FormatUint(uint64(post.ID), 10)+":likes-count") ||
		!redisServer.Exists("mybbs:v1:lru:post:"+strconv.FormatUint(uint64(post.ID), 10)+":comments-count") {
		t.Fatal("post interaction counts were not written to LRU Redis")
	}

	if err := likeRepo.DeleteByUserAndPost(ctx, author.ID, post.ID); err != nil {
		t.Fatalf("delete like directly: %v", err)
	}
	if err := commentRepo.SoftDelete(ctx, comment.ID); err != nil {
		t.Fatalf("delete comment directly: %v", err)
	}
	assertPostCounts(t, svc, post.ID, 1, 1)

	redisServer.FastForward(31 * time.Second)
	assertPostCounts(t, svc, post.ID, 0, 0)

	if err := likeRepo.Create(ctx, &model.PostLike{PostID: post.ID, UserID: author.ID}); err != nil {
		t.Fatalf("recreate like: %v", err)
	}
	if err := commentRepo.Create(ctx, &model.Comment{PostID: post.ID, UserID: author.ID, Content: "fallback"}); err != nil {
		t.Fatalf("recreate comment: %v", err)
	}
	redisServer.Close()
	assertPostCounts(t, svc, post.ID, 1, 1)
}

func assertPostCounts(t *testing.T, svc *service.PostService, postID uint, wantLikes, wantComments int64) {
	t.Helper()
	result, err := svc.GetAllPosts(context.Background(), pagination.Query{PageNo: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetAllPosts: %v", err)
	}
	for _, item := range result.List {
		if item.Post.ID == postID {
			if item.LikeCount != wantLikes || item.CommentCount != wantComments {
				t.Fatalf("post %d counts=(%d,%d), want=(%d,%d)", postID, item.LikeCount, item.CommentCount, wantLikes, wantComments)
			}
			return
		}
	}
	t.Fatalf("post %d was not returned", postID)
}
