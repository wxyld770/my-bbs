package service

import (
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// Register 注册新用户
func (s *UserService) Register(username, password, nickname string) error {
	existing, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		return err
	}
	if existing != nil {
		return bizerr.ErrUsernameExists
	}

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

// Login 登录，验证密码，生成 JWT
func (s *UserService) Login(username, password string) (string, error) {
	user, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", bizerr.ErrLoginFailed
	}

	if !bcrypt.CheckPassword(password, user.Password) {
		return "", bizerr.ErrLoginFailed
	}

	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetMe 获取当前登录用户资料（不含密码）
func (s *UserService) GetMe(userID uint) (*model.User, error) {
	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, bizerr.ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile 更新当前用户的昵称和介绍
func (s *UserService) UpdateProfile(userID uint, nickname, introduction string) error {
	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.ErrUserNotFound
	}
	return s.userRepo.UpdateProfile(userID, nickname, introduction)
}
