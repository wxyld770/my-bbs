package authorization

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"my-bbs/internal/model"
)

var (
	ErrNoAdminUsernames  = errors.New("ADMIN_USERNAMES 至少要包含一个用户名")
	ErrAdminUserNotFound = errors.New("配置的管理员账号不存在")
)

// AdminChecker 是用例层所需的最小管理员身份策略。
type AdminChecker interface {
	IsAdminUsername(username string) bool
}

// AdminUsers 是从配置加载后保持只读的管理员用户名集合。
type AdminUsers struct {
	usernames map[string]struct{}
}

type UserLookup interface {
	FindUserByUsername(ctx context.Context, username string) (*model.User, error)
}

func NewAdminUsers(usernames ...string) *AdminUsers {
	set := make(map[string]struct{}, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username != "" {
			set[username] = struct{}{}
		}
	}
	return &AdminUsers{usernames: set}
}

// ParseAdminUsers 解析逗号分隔的环境变量，例如 admin,moderator。
func ParseAdminUsers(raw string) (*AdminUsers, error) {
	admins := NewAdminUsers(strings.Split(raw, ",")...)
	if len(admins.usernames) == 0 {
		return nil, ErrNoAdminUsernames
	}
	return admins, nil
}

func (a *AdminUsers) IsAdminUsername(username string) bool {
	if a == nil {
		return false
	}
	_, ok := a.usernames[username]
	return ok
}

func (a *AdminUsers) Usernames() []string {
	if a == nil {
		return nil
	}
	result := make([]string, 0, len(a.usernames))
	for username := range a.usernames {
		result = append(result, username)
	}
	sort.Strings(result)
	return result
}

// ValidateExistingAdminUsers 在 HTTP 服务启动前验证配置只引用已有账号。
// 同时二次校验数据库返回的实际用户名，避免大小写/重音 collation 误匹配。
func ValidateExistingAdminUsers(ctx context.Context, users UserLookup, admins *AdminUsers) error {
	if users == nil || admins == nil || len(admins.usernames) == 0 {
		return ErrNoAdminUsernames
	}
	for _, username := range admins.Usernames() {
		user, err := users.FindUserByUsername(ctx, username)
		if err != nil {
			return err
		}
		if user == nil || user.Username != username {
			return fmt.Errorf("%w: %s", ErrAdminUserNotFound, username)
		}
	}
	return nil
}

// IsAdmin 对 nil checker 安全，便于不需要管理员能力的用例测试。
func IsAdmin(checker AdminChecker, username string) bool {
	return checker != nil && checker.IsAdminUsername(username)
}
