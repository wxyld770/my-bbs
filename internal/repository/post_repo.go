package repository

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

// CreatePost 创建帖子
func (r *PostRepository) CreatePost(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

// FindPostByID 根据 ID 查找帖子（不含作者信息）
func (r *PostRepository) FindPostByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	result := r.db.WithContext(ctx).First(&post, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &post, nil
}

// FindPublicPosts 分页查询公开帖子
func (r *PostRepository) FindPublicPosts(ctx context.Context, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("visible = ?", model.VisiblePublic).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

// FindPostsByUserID 分页查询某用户的帖子
func (r *PostRepository) FindPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

// FindPublicPostsByUserID 分页查询某用户的公开帖子
func (r *PostRepository) FindPublicPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND visible = ?", userID, model.VisiblePublic).
		Order("create_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

// UpdatePost 更新帖子
func (r *PostRepository) UpdatePost(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Updates(post).Error
}

// UpdatePostVisible 更新帖子可见性
func (r *PostRepository) UpdatePostVisible(ctx context.Context, id uint, visible uint8) error {
	return r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).
		Update("visible", visible).Error
}

// DeletePost 根据 ID 删除帖子（软删除，写入 deleted 字段）
func (r *PostRepository) DeletePost(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Post{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
