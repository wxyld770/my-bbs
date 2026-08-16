package repository

import (
	"context"
	"errors"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// FindByID 根据 ID 查找评论
func (r *CommentRepository) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	result := r.db.WithContext(ctx).First(&comment, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &comment, nil
}

// FindByPostID 分页查询某帖评论（按创建时间升序，楼层顺序）
func (r *CommentRepository) FindByPostID(ctx context.Context, postID uint, offset, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("create_time ASC").
		Offset(offset).
		Limit(limit).
		Find(&comments).Error
	return comments, err
}

// SoftDelete 软删除评论
func (r *CommentRepository) SoftDelete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Comment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountByPostID 统计某帖评论数
func (r *CommentRepository) CountByPostID(ctx context.Context, postID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}
