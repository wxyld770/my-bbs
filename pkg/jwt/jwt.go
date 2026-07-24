package jwt

import (
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
	"my-bbs/internal/config"
)

type Claims struct {
    UserID uint `json:"user_id"`
    jwt.RegisteredClaims
}

// GenerateToken 生成JWT，有效期24小时
func GenerateToken(userID uint) (string, error) {
    claims := Claims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 过期时间
            IssuedAt:  jwt.NewNumericDate(time.Now()), // 签发时间
            NotBefore: jwt.NewNumericDate(time.Now()), // 生效时间
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(SECRET_KEY())
}

// ParseToken 解析并验证JWT，返回用户ID
func ParseToken(tokenString string) (uint, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
        return SECRET_KEY(), nil
    })
    if err != nil {
        return 0, err
    }
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims.UserID, nil
    }
    return 0, errors.New("invalid token")
}

// 从配置读取secret_key
func SECRET_KEY() string {
	cfg := config.Load()
	return cfg.JWTSecret
}