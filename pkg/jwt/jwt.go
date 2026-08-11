package jwt

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	secretMu        sync.RWMutex
	secretKey       []byte
	ErrTokenExpired = errors.New("令牌已过期")
	ErrTokenInvalid = errors.New("令牌无效")
	ErrSecretEmpty  = errors.New("JWT secret 未初始化")
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// Init 设置签名密钥；应在进程启动且配置校验通过后调用一次。
func Init(secret string) {
	secretMu.Lock()
	defer secretMu.Unlock()
	secretKey = []byte(secret)
}

func getSecretKey() ([]byte, error) {
	secretMu.RLock()
	defer secretMu.RUnlock()
	if len(secretKey) == 0 {
		return nil, ErrSecretEmpty
	}
	return secretKey, nil
}

// GenerateToken 生成JWT，有效期24小时
func GenerateToken(userID uint) (string, error) {
	key, err := getSecretKey()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "my-bbs",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ParseToken 解析并验证JWT，返回用户ID
func ParseToken(tokenString string) (uint, error) {
	key, err := getSecretKey()
	if err != nil {
		return 0, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return key, nil
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

// RefreshToken 用于 token 刷新
func RefreshToken(tokenString string) (string, error) {
	key, err := getSecretKey()
	if err != nil {
		return "", err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
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
