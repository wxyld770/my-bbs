package jwt

import (
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
	"my-bbs/internal/config"
)

var (
    secretKey = SECRET_KEY()
    ErrTokenExpired = errors.New("令牌已过期")
    ErrTokenInvalid = errors.New("令牌无效")
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
            Issuer:    "my-bbs", // 签发人
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(SECRET_KEY())
}

// ParseToken 解析并验证JWT，返回用户ID
func ParseToken(tokenString string) (uint, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // 验证签名算法
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrTokenInvalid
        }
        return secretKey, nil
    })

    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return 0, ErrTokenExpired
        }
        return 0, ErrTokenInvalid
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims.UserID, nil
    }

    return 0, ErrTokenInvalid
}

// 用于token刷新
func RefreshToken(tokenString string) (string, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return secretKey, nil
    })

    if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
        return "", ErrTokenInvalid
    }

    claims, ok := token.Claims.(*Claims)
    if !ok {
        return "", ErrTokenInvalid
    }

    return GenerateToken(claims.UserID)
}

// 从配置读取secret_key
func SECRET_KEY() string {
	cfg := config.Load()
	return cfg.JWTSecret
}