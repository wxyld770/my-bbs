package repository

import (
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
func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// FindByID 根据 ID 查找评论
func (r *CommentRepository) FindByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	result := r.db.First(&comment, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &comment, nil
}

// FindByPostID 分页查询某帖评论（按创建时间升序，楼层顺序）
func (r *CommentRepository) FindByPostID(postID uint, offset, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.
		Where("post_id = ?", postID).
		Order("create_time ASC").
		Offset(offset).
		Limit(limit).
		Find(&comments).Error
	return comments, err
}

// SoftDelete 软删除评论
func (r *CommentRepository) SoftDelete(id uint) error {
	result := r.db.Delete(&model.Comment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountByPostID 统计某帖评论数
func (r *CommentRepository) CountByPostID(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}
