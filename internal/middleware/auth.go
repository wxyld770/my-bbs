package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"my-bbs/internal/authsession"
	"my-bbs/internal/logger"
	"my-bbs/internal/model"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	userIDContextKey         = "userID"
	userStatusContextKey     = "userStatus"
	tokenIDContextKey        = "tokenID"
	tokenExpiresAtContextKey = "tokenExpiresAt"
)

// UserLookup Auth 校验用户状态时使用的最小查询接口
type UserLookup interface {
	FindUserByID(ctx context.Context, id uint) (*model.User, error)
}

// Auth 中间件：验证 JWT Token，校验 Token 未被撤销且用户存在。
// 禁言用户仍可登录和读取；业务写操作由 RequireActiveUser 单独拦截。
// Redis 是撤销校验的必需依赖；缺失或查询失败时请求按 fail-closed 处理。
func Auth(users UserLookup, commands redis.Cmdable) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.ReportError(c, bizerr.ErrTokenMissing)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.ReportError(c, bizerr.ErrTokenFormat)
			return
		}
		if parts[1] == "" {
			response.ReportError(c, bizerr.ErrTokenMissing)
			return
		}

		claims, err := jwt.ParseClaims(parts[1])
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				response.ReportError(c, bizerr.ErrTokenExpired)
				return
			}
			response.ReportError(c, bizerr.ErrInvalidToken)
			return
		}
		revoked, err := authsession.IsRevoked(c.Request.Context(), commands, claims.ID)
		if err != nil {
			response.ReportError(c, errors.Join(bizerr.ErrServiceUnavailable, err))
			return
		}
		if revoked {
			response.ReportError(c, bizerr.ErrInvalidToken)
			return
		}

		user, err := users.FindUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			// 保留 Repository 原始错误，由统一错误中间件记录并安全地返回 500。
			response.ReportError(c, err)
			return
		}
		if user == nil {
			response.ReportError(c, bizerr.ErrUserNotFound)
			return
		}
		logger.Info("authentication succeeded | request_id=%s | user_id=%d", GetRequestID(c), claims.UserID)
		setTokenContext(c, claims)
		c.Set(userStatusContextKey, user.Status)
		c.Next()
	}
}

// RequireActiveUser 将已认证但被禁言的账号限制为只读。
// 必须放在 Auth 之后；上下文缺少状态时按未授权失败，避免误放行。
func RequireActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(userStatusContextKey)
		status, ok := value.(uint)
		if !exists || !ok {
			response.ReportError(c, bizerr.ErrUnauthorized)
			return
		}
		if status != model.UserStatusNormal {
			response.ReportError(c, bizerr.ErrUserMuted)
			return
		}
		c.Next()
	}
}

// OptionalAuth 可选认证中间件：如果有 Token 则解析，没有则继续。
// 不校验禁言状态（仅用于公开读接口附带 is_liked 等）。
func OptionalAuth(commands redis.Cmdable) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		claims, err := jwt.ParseClaims(parts[1])
		if err != nil {
			c.Next()
			return
		}
		revoked, err := authsession.IsRevoked(c.Request.Context(), commands, claims.ID)
		if err != nil {
			response.ReportError(c, errors.Join(bizerr.ErrServiceUnavailable, err))
			return
		}
		if revoked {
			c.Next()
			return
		}
		setTokenContext(c, claims)
		c.Next()
	}
}

// GetUserID 从上下文中获取用户 ID
func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get(userIDContextKey)
	if !exists {
		return 0, false
	}
	userID, ok := val.(uint)
	return userID, ok
}

// GetTokenID 从上下文中获取当前 JWT 的 JTI。
func GetTokenID(c *gin.Context) (string, bool) {
	val, exists := c.Get(tokenIDContextKey)
	if !exists {
		return "", false
	}
	tokenID, ok := val.(string)
	return tokenID, ok && tokenID != ""
}

// GetTokenExpiresAt 从上下文中获取当前 JWT 的到期时间。
func GetTokenExpiresAt(c *gin.Context) (time.Time, bool) {
	val, exists := c.Get(tokenExpiresAtContextKey)
	if !exists {
		return time.Time{}, false
	}
	expiresAt, ok := val.(time.Time)
	return expiresAt, ok && !expiresAt.IsZero()
}

func setTokenContext(c *gin.Context, claims *jwt.Claims) {
	c.Set(userIDContextKey, claims.UserID)
	c.Set(tokenIDContextKey, claims.ID)
	c.Set(tokenExpiresAtContextKey, claims.ExpiresAt.Time)
}
