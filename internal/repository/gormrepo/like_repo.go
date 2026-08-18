package gormrepo

import (
	"context"
	"errors"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) Create(ctx context.Context, like *model.PostLike) error {
	return translateError(r.db.WithContext(ctx).Create(like).Error)
}

func (r *LikeRepository) FindByUserAndPost(ctx context.Context, userID, postID uint) (*model.PostLike, error) {
	var like model.PostLike
	result := r.db.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).First(&like)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &like, nil
}

func (r *LikeRepository) DeleteByUserAndPost(ctx context.Context, userID, postID uint) error {
	return translateDeleteResult(r.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.PostLike{}))
}

func (r *LikeRepository) CountByPostID(ctx context.Context, postID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&count).Error
	return count, translateError(err)
}

func (r *LikeRepository) ExistsByUserAndPost(ctx context.Context, userID, postID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostLike{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Limit(1).
		Count(&count).Error
	return count > 0, translateError(err)
}
