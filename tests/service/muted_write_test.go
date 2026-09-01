package service_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"
)

func TestMutedActorCannotWriteThroughServices(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	users := gormrepo.NewUserRepository(db)
	posts := gormrepo.NewPostRepository(db)
	comments := gormrepo.NewCommentRepository(db)
	likes := gormrepo.NewLikeRepository(db)
	invitations := gormrepo.NewInvitationRepository(db)

	muted := &model.User{Username: "muted-writer", Password: "x", Status: model.UserStatusNormal}
	if err := users.CreateUser(ctx, muted); err != nil {
		t.Fatalf("create muted user: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", muted.ID).Update("status", model.UserStatusMuted).Error; err != nil {
		t.Fatalf("mute user: %v", err)
	}
	post := &model.Post{UserID: muted.ID, Title: "existing", Content: "existing", Visible: model.VisiblePublic}
	if err := posts.CreatePost(ctx, post); err != nil {
		t.Fatalf("create existing post: %v", err)
	}
	comment := &model.Comment{PostID: post.ID, UserID: muted.ID, Content: "existing"}
	if err := comments.Create(ctx, comment); err != nil {
		t.Fatalf("create existing comment: %v", err)
	}

	postService := service.NewPostService(posts, users, comments, likes, authorization.NewAdminUsers(muted.Username))
	commentService := service.NewCommentService(comments, posts, users)
	likeService := service.NewLikeService(likes, posts, users)
	userService := service.NewUserServiceWithInvitations(users, invitations, authorization.NewAdminUsers(muted.Username))
	newTitle := "blocked"

	tests := []struct {
		name string
		call func() error
	}{
		{name: "create post", call: func() error { return postService.CreatePost(ctx, muted.ID, "blocked", "blocked") }},
		{name: "update post", call: func() error { return postService.UpdatePost(ctx, post.ID, muted.ID, &newTitle, nil) }},
		{name: "delete post", call: func() error { return postService.DeletePost(ctx, post.ID, muted.ID) }},
		{name: "pin post", call: func() error {
			_, err := postService.PinPost(ctx, post.ID, muted.ID, model.PostPinDurationPermanent)
			return err
		}},
		{name: "unpin post", call: func() error { return postService.UnpinPost(ctx, post.ID, muted.ID) }},
		{name: "set visibility", call: func() error { return postService.SetPostVisible(ctx, post.ID, muted.ID, model.VisiblePrivate) }},
		{name: "create comment", call: func() error { return commentService.CreateComment(ctx, post.ID, muted.ID, "blocked") }},
		{name: "delete comment", call: func() error { return commentService.DeleteComment(ctx, comment.ID, muted.ID) }},
		{name: "toggle like", call: func() error {
			_, err := likeService.Toggle(ctx, post.ID, muted.ID)
			return err
		}},
		{name: "update profile", call: func() error { return userService.UpdateProfile(ctx, muted.ID, "blocked", "blocked") }},
		{name: "update avatar", call: func() error { return userService.UpdateAvatar(ctx, muted.ID, "https://example.com/blocked.png") }},
		{name: "generate invitation", call: func() error {
			_, err := userService.GenerateInvitation(ctx, muted.ID)
			return err
		}},
		{name: "manage user", call: func() error { return userService.SetUserStatus(ctx, muted.ID, muted.ID+1, model.UserStatusMuted) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, bizerr.ErrUserMuted) {
				t.Fatalf("error=%v, want ErrUserMuted", err)
			}
		})
	}
}
