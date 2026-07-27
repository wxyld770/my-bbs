package middleware

import (
    "net/http"
    "strings"
    "my-bbs/pkg/jwt"
    "github.com/gin-gonic/gin"
)

// Auth 中间件：验证 JWT Token，将用户 ID 注入上下文
func Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从请求头获取 Authorization
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "缺少认证令牌",
            })
            return
        }

        // 2. 验证格式：必须是 Bearer <token>
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "认证令牌格式错误，请使用 Bearer <token>",
            })
            return
        }

        tokenString := parts[1]
        if tokenString == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "认证令牌不能为空",
            })
            return
        }

        // 3. 解析 Token
        userId, err := jwt.ParseToken(tokenString)
        if err != nil {
            // 根据错误类型给出更具体的提示
            if err == jwt.ErrTokenExpired {
                c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                    "error": "认证令牌已过期，请重新登录",
                })
                return
            }
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "无效的认证令牌",
            })
            return
        }

        // 4. 将用户 ID 注入 Context，供后续 Handler 使用
        c.Set("userId", userId)
        
        // 5. 继续处理请求
        c.Next()
    }
}

// OptionalAuth 可选认证中间件：如果有 Token 则解析，没有则继续
// 适用于“登录后可看更多内容，但未登录也能看”的场景
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
            c.Set("userId", userID)
        }
        // 解析失败也继续，不阻断请求
        c.Next()
    }
}

// GetUserID 从上下文中获取用户 ID（辅助函数，方便 Handler 使用）
func GetUserID(c *gin.Context) (uint, bool) {
    val, exists := c.Get("userId")
    if !exists {
        return 0, false
    }
    userId, ok := val.(uint)
    return userId, ok
}