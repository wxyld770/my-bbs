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
    result := r.db.Create(user)
    if result.Error != nil {
        return result.Error
    }
    return nil
}

// FindUserByUsername 根据用户名查找用户（用于登录/注册校验）
func (r *UserRepository) FindUserByUsername(username string) (*model.User, error) {
    var user model.User
    result := r.db.Where("username = ?", username).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, nil // 未找到返回 nil，不返回错误
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

// UpdatePost 更新用户信息
func (r *UserRepository) UpdateUser(user *model.User) error {
    result := r.db.Updates(user)
    return result.Error
}