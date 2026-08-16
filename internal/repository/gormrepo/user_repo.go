package gormrepo

import (
	"context"
	"errors"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造 GORM 用户仓储实现。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	return translateError(r.db.WithContext(ctx).Create(user).Error)
}

func (r *UserRepository) FindUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &user, nil
}

func (r *UserRepository) FindUsersByIDs(ctx context.Context, ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, translateError(err)
	}
	return users, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	return translateError(r.db.WithContext(ctx).Updates(user).Error)
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id uint, nickname, introduction string) error {
	return translateError(r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Select("nickname", "introduction").
		Updates(map[string]any{
			"nickname":     nickname,
			"introduction": introduction,
		}).Error)
}
