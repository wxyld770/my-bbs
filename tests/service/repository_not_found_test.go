package service_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
)

func TestUserService_MapsMutationNotFound(t *testing.T) {
	svc := service.NewUserService(&mutationNotFoundUserRepository{})

	err := svc.UpdateProfile(context.Background(), 7, "nickname", "introduction")
	if !errors.Is(err, bizerr.ErrUserNotFound) {
		t.Fatalf("UpdateProfile() error = %v, want ErrUserNotFound", err)
	}
}

func TestPostService_MapsMutationNotFound(t *testing.T) {
	const userID = 7
	repo := &mutationNotFoundPostRepository{post: &model.Post{
		BaseModel: model.BaseModel{ID: 9},
		UserID:    userID,
		Title:     "title",
		Content:   "content",
		Visible:   model.VisiblePublic,
	}}
	svc := service.NewPostService(repo, nil, nil, nil)
	title := "new title"

	tests := []struct {
		name string
		call func() error
	}{
		{name: "update", call: func() error {
			return svc.UpdatePost(context.Background(), repo.post.ID, userID, &title, nil)
		}},
		{name: "delete", call: func() error {
			return svc.DeletePost(context.Background(), repo.post.ID, userID)
		}},
		{name: "set visibility", call: func() error {
			return svc.SetPostVisible(context.Background(), repo.post.ID, userID, model.VisiblePrivate)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, bizerr.ErrPostNotFound) {
				t.Fatalf("service error = %v, want ErrPostNotFound", err)
			}
		})
	}
}

func TestCommentService_MapsMutationNotFound(t *testing.T) {
	const userID = 7
	svc := service.NewCommentService(&mutationNotFoundCommentRepository{comment: &model.Comment{
		BaseModel: model.BaseModel{ID: 11},
		UserID:    userID,
	}}, nil, nil)

	err := svc.DeleteComment(context.Background(), 11, userID)
	if !errors.Is(err, bizerr.ErrCommentNotFound) {
		t.Fatalf("DeleteComment() error = %v, want ErrCommentNotFound", err)
	}
}

func TestLikeService_TreatsConcurrentMissingLikeAsUnliked(t *testing.T) {
	const (
		postID = 9
		userID = 7
	)
	svc := service.NewLikeService(
		&concurrentlyDeletedLikeRepository{like: &model.PostLike{PostID: postID, UserID: userID}},
		&publicPostReader{post: &model.Post{BaseModel: model.BaseModel{ID: postID}, Visible: model.VisiblePublic}},
	)

	result, err := svc.Toggle(context.Background(), postID, userID)
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}
	if result.Liked || result.LikeCount != 0 {
		t.Fatalf("Toggle() result = %+v, want unliked with zero count", result)
	}
}

type mutationNotFoundUserRepository struct{}

func (*mutationNotFoundUserRepository) CreateUser(context.Context, *model.User) error { return nil }
func (*mutationNotFoundUserRepository) FindUserByUsername(context.Context, string) (*model.User, error) {
	return nil, nil
}
func (*mutationNotFoundUserRepository) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	return &model.User{BaseModel: model.BaseModel{ID: id}}, nil
}
func (*mutationNotFoundUserRepository) UpdateProfile(context.Context, uint, string, string) error {
	return repository.ErrNotFound
}

type mutationNotFoundPostRepository struct {
	post *model.Post
}

func (*mutationNotFoundPostRepository) CreatePost(context.Context, *model.Post) error { return nil }
func (r *mutationNotFoundPostRepository) FindPostByID(context.Context, uint) (*model.Post, error) {
	return r.post, nil
}
func (*mutationNotFoundPostRepository) FindPublicPosts(context.Context, int, int) ([]model.Post, error) {
	return nil, nil
}
func (*mutationNotFoundPostRepository) FindPostsByUserID(context.Context, uint, int, int) ([]model.Post, error) {
	return nil, nil
}
func (*mutationNotFoundPostRepository) FindPublicPostsByUserID(context.Context, uint, int, int) ([]model.Post, error) {
	return nil, nil
}
func (*mutationNotFoundPostRepository) UpdatePost(context.Context, *model.Post) error {
	return repository.ErrNotFound
}
func (*mutationNotFoundPostRepository) UpdatePostVisible(context.Context, uint, uint8) error {
	return repository.ErrNotFound
}
func (*mutationNotFoundPostRepository) DeletePost(context.Context, uint) error {
	return repository.ErrNotFound
}

type mutationNotFoundCommentRepository struct {
	comment *model.Comment
}

func (*mutationNotFoundCommentRepository) CountByPostID(context.Context, uint) (int64, error) {
	return 0, nil
}
func (*mutationNotFoundCommentRepository) Create(context.Context, *model.Comment) error { return nil }
func (r *mutationNotFoundCommentRepository) FindByID(context.Context, uint) (*model.Comment, error) {
	return r.comment, nil
}
func (*mutationNotFoundCommentRepository) FindByPostID(context.Context, uint, int, int) ([]model.Comment, error) {
	return nil, nil
}
func (*mutationNotFoundCommentRepository) SoftDelete(context.Context, uint) error {
	return repository.ErrNotFound
}

type concurrentlyDeletedLikeRepository struct {
	like *model.PostLike
}

func (*concurrentlyDeletedLikeRepository) CountByPostID(context.Context, uint) (int64, error) {
	return 0, nil
}
func (*concurrentlyDeletedLikeRepository) ExistsByUserAndPost(context.Context, uint, uint) (bool, error) {
	return false, nil
}
func (*concurrentlyDeletedLikeRepository) Create(context.Context, *model.PostLike) error { return nil }
func (r *concurrentlyDeletedLikeRepository) FindByUserAndPost(context.Context, uint, uint) (*model.PostLike, error) {
	return r.like, nil
}
func (*concurrentlyDeletedLikeRepository) DeleteByUserAndPost(context.Context, uint, uint) error {
	return repository.ErrNotFound
}

type publicPostReader struct {
	post *model.Post
}

func (r *publicPostReader) FindPostByID(context.Context, uint) (*model.Post, error) {
	return r.post, nil
}
