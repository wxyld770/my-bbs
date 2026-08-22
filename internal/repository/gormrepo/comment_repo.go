package gormrepo

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

func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	return translateError(r.db.WithContext(ctx).Create(comment).Error)
}

func (r *CommentRepository) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	result := r.db.WithContext(ctx).First(&comment, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &comment, nil
}

func (r *CommentRepository) FindByPostID(ctx context.Context, postID uint, offset, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("create_time ASC").
		Offset(offset).
		Limit(limit).
		Find(&comments).Error
	return comments, translateError(err)
}

func (r *CommentRepository) SoftDelete(ctx context.Context, id uint) error {
	return translateDeleteResult(r.db.WithContext(ctx).Delete(&model.Comment{}, id))
}

func (r *CommentRepository) CountByPostID(ctx context.Context, postID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	return count, translateError(err)
}

func (r *CommentRepository) CountByPostIDs(ctx context.Context, postIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(postIDs))
	if len(postIDs) == 0 {
		return counts, nil
	}

	var rows []struct {
		PostID uint  `gorm:"column:post_id"`
		Total  int64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("post_id, COUNT(*) AS total").
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Scan(&rows).Error
	if err != nil {
		return nil, translateError(err)
	}
	for _, row := range rows {
		counts[row.PostID] = row.Total
	}
	return counts, nil
}
