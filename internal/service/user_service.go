package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"my-bbs/internal/authorization"
	"my-bbs/internal/authsession"
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	userRepo repository.UserRepository
	redis    redis.Cmdable
	admins   authorization.AdminChecker
}

func NewUserService(userRepo repository.UserRepository, adminCheckers ...authorization.AdminChecker) *UserService {
	return &UserService{userRepo: userRepo, admins: firstAdminChecker(adminCheckers)}
}

// NewUserServiceWithRedis 为 Token 撤销注入 Redis 命令接口。
func NewUserServiceWithRedis(
	userRepo repository.UserRepository,
	redisCommands redis.Cmdable,
	adminCheckers ...authorization.AdminChecker,
) *UserService {
	return &UserService{userRepo: userRepo, redis: redisCommands, admins: firstAdminChecker(adminCheckers)}
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

// Logout 撤销当前 JWT，撤销记录只保留到 Token 原始到期时间。
// 已到期 Token 无需写 Redis，按幂等成功处理。
func (s *UserService) Logout(ctx context.Context, tokenID string, expiresAt time.Time) error {
	if tokenID == "" || expiresAt.IsZero() {
		return bizerr.ErrInvalidToken
	}
	if err := authsession.Revoke(ctx, s.redis, tokenID, expiresAt); err != nil {
		return errors.Join(bizerr.ErrServiceUnavailable, err)
	}
	return nil
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

// SetUserStatus 允许配置中的管理员禁言或解禁普通用户。
func (s *UserService) SetUserStatus(ctx context.Context, actorID, targetID uint, status uint) error {
	if status != model.UserStatusMuted && status != model.UserStatusNormal {
		return bizerr.ErrBadRequest.WithMessage("用户状态无效")
	}

	actor, err := s.userRepo.FindUserByID(ctx, actorID)
	if err != nil {
		return err
	}
	if actor == nil {
		return bizerr.ErrUserNotFound
	}
	if !actor.IsActive() {
		return bizerr.ErrUserMuted
	}
	if !authorization.IsAdmin(s.admins, actor.Username) {
		return bizerr.ErrForbidden
	}

	target, err := s.userRepo.FindUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target == nil {
		return bizerr.ErrUserNotFound
	}
	if authorization.IsAdmin(s.admins, target.Username) {
		return bizerr.ErrAdminCannotManageAdmin
	}
	if err := s.userRepo.UpdateUserStatus(ctx, targetID, status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return bizerr.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *UserService) IsAdminUsername(username string) bool {
	return authorization.IsAdmin(s.admins, username)
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

func firstAdminChecker(checkers []authorization.AdminChecker) authorization.AdminChecker {
	if len(checkers) == 0 {
		return nil
	}
	return checkers[0]
}
