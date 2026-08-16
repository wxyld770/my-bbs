package middleware

import (
	"context"
	"strings"

	"my-bbs/internal/logger"
	"my-bbs/internal/model"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserLookup Auth 校验用户状态时使用的最小查询接口
type UserLookup interface {
	FindUserByID(ctx context.Context, id uint) (*model.User, error)
}

// Auth 中间件：验证 JWT Token，校验用户存在且未禁言，将用户 ID 注入上下文
func Auth(users UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AbortFail(c, bizerr.ErrTokenMissing)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.AbortFail(c, bizerr.ErrTokenFormat)
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			response.AbortFail(c, bizerr.ErrTokenMissing)
			return
		}

		userID, err := jwt.ParseToken(tokenString)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				response.AbortFail(c, bizerr.ErrTokenExpired)
				return
			}
			response.AbortFail(c, bizerr.ErrInvalidToken)
			return
		}

		user, err := users.FindUserByID(c.Request.Context(), userID)
		if err != nil {
			response.AbortFail(c, bizerr.ErrInternal)
			return
		}
		if user == nil {
			response.AbortFail(c, bizerr.ErrUserNotFound)
			return
		}
		if !user.IsActive() {
			response.AbortFail(c, bizerr.ErrUserMuted)
			return
		}

		logger.Info("get userID: %d from Authorization", userID)
		c.Set("userID", userID)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件：如果有 Token 则解析，没有则继续。
// 不校验禁言状态（仅用于公开读接口附带 is_liked 等）。
func OptionalAuth() gin.HandlerFunc {
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

		userID, err := jwt.ParseToken(parts[1])
		if err == nil {
			c.Set("userID", userID)
		}
		c.Next()
	}
}

// GetUserID 从上下文中获取用户 ID
func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	userID, ok := val.(uint)
	return userID, ok
}
