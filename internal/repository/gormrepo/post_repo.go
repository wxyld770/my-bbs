package gormrepo

import (
	"context"
	"errors"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) CreatePost(ctx context.Context, post *model.Post) error {
	return translateError(r.db.WithContext(ctx).Create(post).Error)
}

func (r *PostRepository) FindPostByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	result := r.db.WithContext(ctx).First(&post, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &post, nil
}

func (r *PostRepository) FindPublicPosts(ctx context.Context, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("visible = ?", model.VisiblePublic).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) FindPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) FindPublicPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND visible = ?", userID, model.VisiblePublic).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) UpdatePost(ctx context.Context, post *model.Post) error {
	return translateError(r.db.WithContext(ctx).Updates(post).Error)
}

func (r *PostRepository) UpdatePostVisible(ctx context.Context, id uint, visible uint8) error {
	return translateError(r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).
		Update("visible", visible).Error)
}

func (r *PostRepository) DeletePost(ctx context.Context, id uint) error {
	return translateDeleteResult(r.db.WithContext(ctx).Delete(&model.Post{}, id))
}
