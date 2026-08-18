package service

import (
	"context"
	"errors"
	"strings"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// Register 注册新用户
func (s *UserService) Register(ctx context.Context, username, password, nickname string) error {
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	if err := validateRuneLength(username, "用户名", 3, 64); err != nil {
		return err
	}
	if err := validateRuneLength(password, "密码", 6, 64); err != nil {
		return err
	}
	// bcrypt 只接受最多 72 字节；字符数校验无法覆盖多字节密码。
	if err := validateByteLength(password, "密码", 72); err != nil {
		return err
	}
	if err := validateRuneLength(nickname, "昵称", 0, 64); err != nil {
		return err
	}

	existing, err := s.userRepo.FindUserByUsername(ctx, username)
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
		Status:   model.UserStatusNormal,
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return bizerr.ErrUsernameExists
		}
		return err
	}
	return nil
}

// Login 登录，验证密码与账号状态，生成 JWT
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	user, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", bizerr.ErrLoginFailed
	}

	if !bcrypt.CheckPassword(password, user.Password) {
		return "", bizerr.ErrLoginFailed
	}

	if !user.IsActive() {
		return "", bizerr.ErrUserMuted
	}

	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetMe 获取当前登录用户资料（不含密码）
func (s *UserService) GetMe(ctx context.Context, userID uint) (*model.User, error) {
	return s.getUserOrNotFound(ctx, userID)
}

// GetPublicProfile 获取公开用户资料
func (s *UserService) GetPublicProfile(ctx context.Context, userID uint) (*model.User, error) {
	return s.getUserOrNotFound(ctx, userID)
}

// UpdateProfile 更新当前用户的昵称和介绍
func (s *UserService) UpdateProfile(ctx context.Context, userID uint, nickname, introduction string) error {
	nickname = strings.TrimSpace(nickname)
	introduction = strings.TrimSpace(introduction)
	if err := validateRuneLength(nickname, "昵称", 0, 64); err != nil {
		return err
	}
	if err := validateRuneLength(introduction, "个人介绍", 0, 1024); err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.ErrUserNotFound
	}
	if err := s.userRepo.UpdateProfile(ctx, userID, nickname, introduction); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return bizerr.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *UserService) getUserOrNotFound(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, bizerr.ErrUserNotFound
	}
	return user, nil
}
