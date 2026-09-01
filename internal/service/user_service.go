package service

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"net/url"
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
	userRepo       repository.UserRepository
	invitationRepo repository.InvitationRepository
	redis          redis.Cmdable
	admins         authorization.AdminChecker
}

func NewUserService(userRepo repository.UserRepository, adminCheckers ...authorization.AdminChecker) *UserService {
	return &UserService{userRepo: userRepo, admins: firstAdminChecker(adminCheckers)}
}

// NewUserServiceWithInvitations 为注册和邀请码生成注入邀请码仓储。
func NewUserServiceWithInvitations(
	userRepo repository.UserRepository,
	invitationRepo repository.InvitationRepository,
	adminCheckers ...authorization.AdminChecker,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		invitationRepo: invitationRepo,
		admins:         firstAdminChecker(adminCheckers),
	}
}

// NewUserServiceWithRedis 为 Token 撤销注入 Redis 命令接口。
func NewUserServiceWithRedis(
	userRepo repository.UserRepository,
	redisCommands redis.Cmdable,
	adminCheckers ...authorization.AdminChecker,
) *UserService {
	return &UserService{userRepo: userRepo, redis: redisCommands, admins: firstAdminChecker(adminCheckers)}
}

// NewUserServiceWithRedisAndInvitations 注入用户模块的全部生产依赖。
func NewUserServiceWithRedisAndInvitations(
	userRepo repository.UserRepository,
	invitationRepo repository.InvitationRepository,
	redisCommands redis.Cmdable,
	adminCheckers ...authorization.AdminChecker,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		invitationRepo: invitationRepo,
		redis:          redisCommands,
		admins:         firstAdminChecker(adminCheckers),
	}
}

// Register 注册新用户
func (s *UserService) Register(
	ctx context.Context,
	username, password, nickname, inviteCode string,
) error {
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
	inviteCode, err := validateInvitationCode(inviteCode)
	if err != nil {
		return err
	}
	if s.invitationRepo == nil {
		return fmt.Errorf("invitation repository is required for registration")
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
	if err := s.invitationRepo.RegisterUserWithInvitation(ctx, user, inviteCode); err != nil {
		if errors.Is(err, repository.ErrInvitationUnavailable) {
			return bizerr.ErrInvitationUnavailable
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			return bizerr.ErrUsernameExists
		}
		return err
	}
	return nil
}

const (
	invitationCodeLength      = 6
	invitationCodeCreateTries = 10
	invitationCodeAlphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	InvitationNewUserPeriod   = 7 * 24 * time.Hour
	avatarURLMaxBytes         = 2048
	AvatarUpdateInterval      = 24 * time.Hour
)

// GenerateInvitation 为当前用户生成一个新的单次使用邀请码。
// 明文邀请码只由当前调用返回；服务端不提供历史查询接口。
func (s *UserService) GenerateInvitation(ctx context.Context, creatorID uint) (string, error) {
	if creatorID == 0 {
		return "", bizerr.ErrUnauthorized
	}
	if s.invitationRepo == nil {
		return "", fmt.Errorf("invitation repository is required for invitation generation")
	}

	creator, err := s.userRepo.FindUserByID(ctx, creatorID)
	if err != nil {
		return "", err
	}
	if creator == nil {
		return "", bizerr.ErrUserNotFound
	}
	if !creator.IsActive() {
		return "", bizerr.ErrUserMuted
	}
	if invitationRequiresPublishedPost(creator.CreateTime, time.Now()) {
		hasPublishedPost, err := s.invitationRepo.HasCreatorEverPublishedPost(ctx, creatorID)
		if err != nil {
			return "", err
		}
		if !hasPublishedPost {
			return "", bizerr.ErrInvitationGenerationRestricted
		}
	}

	for range invitationCodeCreateTries {
		code, err := newInvitationCode()
		if err != nil {
			return "", fmt.Errorf("generate invitation code: %w", err)
		}
		invitation := &model.Invitation{Code: code, CreatorID: creatorID}
		if err := s.invitationRepo.CreateInvitation(ctx, invitation); err != nil {
			if errors.Is(err, repository.ErrAlreadyExists) {
				continue
			}
			return "", err
		}
		return code, nil
	}

	return "", fmt.Errorf("generate a unique invitation code after %d attempts", invitationCodeCreateTries)
}

func invitationRequiresPublishedPost(createdAt, now time.Time) bool {
	return !createdAt.IsZero() && now.Before(createdAt.Add(InvitationNewUserPeriod))
}

func validateInvitationCode(inviteCode string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(inviteCode))
	if code == "" {
		return "", bizerr.ErrInvitationRequired
	}
	if len(code) != invitationCodeLength {
		return "", bizerr.ErrInvitationUnavailable
	}
	for i := range len(code) {
		char := code[i]
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return "", bizerr.ErrInvitationUnavailable
		}
	}
	return code, nil
}

// newInvitationCode 使用拒绝采样避免直接取模带来的轻微分布偏差。
func newInvitationCode() (string, error) {
	code := make([]byte, invitationCodeLength)
	const unbiasedLimit = 252 // 36 * 7，是不超过 256 的最大 36 倍数。
	for i := range code {
		for {
			var randomByte [1]byte
			if _, err := cryptorand.Read(randomByte[:]); err != nil {
				return "", err
			}
			if randomByte[0] >= unbiasedLimit {
				continue
			}
			code[i] = invitationCodeAlphabet[int(randomByte[0])%len(invitationCodeAlphabet)]
			break
		}
	}
	return string(code), nil
}

// Login 登录并生成 JWT。禁言账号可以登录并浏览，但写操作会被统一限制。
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
	if _, err := requireActiveActor(ctx, s.userRepo, userID); err != nil {
		return err
	}

	if err := s.userRepo.UpdateProfile(ctx, userID, nickname, introduction); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return bizerr.ErrUserNotFound
		}
		return err
	}
	return nil
}

// UpdateAvatar 更新当前用户的头像链接。空链接用于恢复默认头像；真正发生
// 变化的两次更新至少间隔 24 小时，最终限额由 Repository 条件更新原子保证。
func (s *UserService) UpdateAvatar(ctx context.Context, userID uint, rawAvatarURL string) error {
	if userID == 0 {
		return bizerr.ErrUnauthorized
	}
	avatarURL, err := normalizeAvatarURL(rawAvatarURL)
	if err != nil {
		return err
	}
	user, err := requireActiveActor(ctx, s.userRepo, userID)
	if err != nil {
		return err
	}

	if user.AvatarURL == avatarURL {
		return nil
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	if user.AvatarUpdatedAt != nil && now.Before(user.AvatarUpdatedAt.Add(AvatarUpdateInterval)) {
		return bizerr.ErrAvatarUpdateTooFrequent
	}
	if err := s.userRepo.UpdateAvatar(
		ctx,
		userID,
		avatarURL,
		now,
		now.Add(-AvatarUpdateInterval),
	); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return bizerr.ErrUserNotFound
		case errors.Is(err, repository.ErrAvatarUpdateTooFrequent):
			return bizerr.ErrAvatarUpdateTooFrequent
		default:
			return err
		}
	}
	return nil
}

func normalizeAvatarURL(rawAvatarURL string) (string, error) {
	avatarURL := strings.TrimSpace(rawAvatarURL)
	if avatarURL == "" {
		return "", nil
	}
	if len(avatarURL) > avatarURLMaxBytes {
		return "", bizerr.ErrInvalidAvatarURL
	}
	parsed, err := url.ParseRequestURI(avatarURL)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return "", bizerr.ErrInvalidAvatarURL
	}
	parsed.Scheme = "https"
	return parsed.String(), nil
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
