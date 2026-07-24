package service

import (
    "errors"
    "my-bbs/internal/model"
    "my-bbs/internal/repository"
    "my-bbs/pkg/bcrypt"
    "my-bbs/pkg/jwt"
)

type UserService struct {
    userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
    return &UserService{userRepo: userRepo}
}

// Register 注册新用户
// 返回 error，成功则返回 nil
func (s *UserService) Register(username, password, nickname string) error {
    // 检查用户名是否已存在
    existing, err := s.userRepo.FindUserByUsername(username)
    if err != nil {
        return err
    }
    if existing != nil {
        return errors.New("用户名已存在")
    }

    // 加密密码
    hashed, err := bcrypt.HashPassword(password)
    if err != nil {
        return err
    }

    user := &model.User{
        Username: username,
        Password: hashed,
        Nickname: nickname,
    }
    return s.userRepo.CreateUser(user)
}

// Login 登录，验证密码，生成JWT
// 返回 token 和 error
func (s *UserService) Login(username, password string) (string, error) {
    user, err := s.userRepo.FindUserByUsername(username)
    if err != nil {
        return "", err
    }
    if user == nil {
        return "", errors.New("用户名或密码错误")
    }

    // 验证密码
    if !bcrypt.CheckPassword(password, user.Password) {
        return "", errors.New("用户名或密码错误")
    }

    // 生成JWT
    token, err := jwt.GenerateToken(user.ID)
    if err != nil {
        return "", err
    }
    return token, nil
}