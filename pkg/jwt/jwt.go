package jwt

import (
	"crypto/rand"
	"encoding/base64"
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

const tokenIssuer = "my-bbs"

type Claims struct {
	UserID         uint   `json:"user_id"`
	SessionVersion uint64 `json:"session_version"`
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

// GenerateToken 生成 JWT，有效期 24 小时。每个 Token 都有独立的随机 JTI，
// 用于精确撤销当前 Token，而不影响同一用户的其他登录会话。
func GenerateToken(userID uint) (string, error) {
	return GenerateTokenWithSessionVersion(userID, 0)
}

// GenerateTokenWithSessionVersion 生成绑定用户当前会话版本的 JWT。
// 密码变更递增版本后，此前签发的 Token 会在认证边界被统一拒绝。
func GenerateTokenWithSessionVersion(userID uint, sessionVersion uint64) (string, error) {
	key, err := getSecretKey()
	if err != nil {
		return "", err
	}
	tokenID, err := generateTokenID()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		UserID:         userID,
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    tokenIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ParseClaims 解析并验证 JWT，返回认证和 Token 撤销所需的完整 Claims。
func ParseClaims(tokenString string) (*Claims, error) {
	key, err := getSecretKey()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(_ *jwt.Token) (interface{}, error) { return key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid || claims.ID == "" || claims.ExpiresAt == nil {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// ParseToken 解析并验证 JWT，返回用户 ID。
// 保留该 API 以兼容现有调用方；新的认证边界应使用 ParseClaims。
func ParseToken(tokenString string) (uint, error) {
	claims, err := ParseClaims(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func generateTokenID() (string, error) {
	var tokenID [16]byte
	if _, err := rand.Read(tokenID[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenID[:]), nil
}
