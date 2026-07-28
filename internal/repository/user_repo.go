package repository

import (
	"errors"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造函数，依赖注入 DB
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

// FindUserByUsername 根据用户名查找用户（用于登录/注册校验）
func (r *UserRepository) FindUserByUsername(username string) (*model.User, error) {
	var user model.User
	result := r.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindUserByID 根据 ID 查找用户
func (r *UserRepository) FindUserByID(id uint) (*model.User, error) {
	var user model.User
	result := r.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindUsersByIDs 根据 ID 列表批量查找用户
func (r *UserRepository) FindUsersByIDs(ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(user *model.User) error {
	return r.db.Updates(user).Error
}

// UpdateProfile 更新用户昵称和介绍（允许设为空字符串）
func (r *UserRepository) UpdateProfile(id uint, nickname, introduction string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Select("nickname", "introduction").
		Updates(map[string]any{
			"nickname":     nickname,
			"introduction": introduction,
		}).Error
}
