package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-bbs/internal/authsession"
	"my-bbs/internal/middleware"
	"my-bbs/internal/model"
	"my-bbs/pkg/bizerr"
	jwtpkg "my-bbs/pkg/jwt"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type activeUserLookup struct{}

func (activeUserLookup) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	return &model.User{
		BaseModel: model.BaseModel{ID: id},
		Status:    model.UserStatusNormal,
	}, nil
}

type mutedUserLookup struct{}

func (mutedUserLookup) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	return &model.User{
		BaseModel: model.BaseModel{ID: id},
		Status:    model.UserStatusMuted,
	}, nil
}

type versionedUserLookup struct {
	version uint64
}

func (lookup versionedUserLookup) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	return &model.User{
		BaseModel:      model.BaseModel{ID: id},
		Status:         model.UserStatusNormal,
		SessionVersion: lookup.version,
	}, nil
}

func TestAuthAllowsMutedReadsAndActiveGuardRejectsWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("muted-read-only-middleware-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	token, err := jwtpkg.GenerateToken(15)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	router := gin.New()
	router.Use(middleware.ErrorHandler(), middleware.Auth(mutedUserLookup{}, client))
	router.GET("/read", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/write", middleware.RequireActiveUser(), func(c *gin.Context) { c.Status(http.StatusOK) })

	readRequest := httptest.NewRequest(http.MethodGet, "/read", nil)
	readRequest.Header.Set("Authorization", "Bearer "+token)
	readResponse := httptest.NewRecorder()
	router.ServeHTTP(readResponse, readRequest)
	if readResponse.Code != http.StatusOK {
		t.Fatalf("muted read status = %d, want 200; body=%s", readResponse.Code, readResponse.Body.String())
	}

	writeRequest := httptest.NewRequest(http.MethodPost, "/write", nil)
	writeRequest.Header.Set("Authorization", "Bearer "+token)
	writeResponse := httptest.NewRecorder()
	router.ServeHTTP(writeResponse, writeRequest)
	assertBusinessError(t, writeResponse, bizerr.ErrUserMuted)
}

func TestAuthPublishesTokenClaimsAndRejectsRevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("auth-middleware-public-contract-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	token, err := jwtpkg.GenerateToken(7)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := jwtpkg.ParseClaims(token)
	if err != nil {
		t.Fatalf("ParseClaims() error = %v", err)
	}

	router := newAuthRouter(client)
	response := performAuthenticatedRequest(router, token)
	if response.Code != http.StatusOK {
		t.Fatalf("valid token status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var body struct {
		UserID  uint   `json:"user_id"`
		TokenID string `json:"token_id"`
		HasExp  bool   `json:"has_exp"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UserID != 7 || body.TokenID != claims.ID || !body.HasExp {
		t.Fatalf("auth context = %+v, want user=7 token=%q and expiry", body, claims.ID)
	}

	if err := authsession.Revoke(context.Background(), client, claims.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed revoked token: %v", err)
	}
	response = performAuthenticatedRequest(router, token)
	assertBusinessError(t, response, bizerr.ErrInvalidToken)
}

func TestAuthAndOptionalAuthFailClosedWhenRedisFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("auth-fail-closed-public-contract-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	token, err := jwtpkg.GenerateToken(9)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	authResponse := performAuthenticatedRequest(newAuthRouter(client), token)
	assertBusinessError(t, authResponse, bizerr.ErrServiceUnavailable)

	optionalRouter := gin.New()
	optionalRouter.Use(middleware.ErrorHandler(), middleware.OptionalAuth(activeUserLookup{}, client))
	optionalRouter.GET("/private", func(c *gin.Context) { c.Status(http.StatusOK) })
	optionalResponse := performAuthenticatedRequest(optionalRouter, token)
	assertBusinessError(t, optionalResponse, bizerr.ErrServiceUnavailable)
}

func TestOptionalAuthTreatsRevokedTokenAsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("optional-auth-public-contract-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	token, err := jwtpkg.GenerateToken(11)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := jwtpkg.ParseClaims(token)
	if err != nil {
		t.Fatalf("ParseClaims() error = %v", err)
	}
	if err := authsession.Revoke(context.Background(), client, claims.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed revoked token: %v", err)
	}

	router := gin.New()
	router.Use(middleware.ErrorHandler(), middleware.OptionalAuth(activeUserLookup{}, client))
	router.GET("/private", func(c *gin.Context) {
		_, authenticated := middleware.GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": authenticated})
	})
	response := performAuthenticatedRequest(router, token)
	if response.Code != http.StatusOK || response.Body.String() != "{\"authenticated\":false}" {
		t.Fatalf("optional auth response = %d %s", response.Code, response.Body.String())
	}
}

func TestAuthRejectsStaleSessionVersionAndOptionalAuthTreatsItAsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("auth-session-version-public-contract-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	token, err := jwtpkg.GenerateTokenWithSessionVersion(23, 4)
	if err != nil {
		t.Fatalf("GenerateTokenWithSessionVersion() error = %v", err)
	}

	authRouter := gin.New()
	authRouter.Use(middleware.ErrorHandler(), middleware.Auth(versionedUserLookup{version: 5}, client))
	authRouter.GET("/private", func(c *gin.Context) { c.Status(http.StatusOK) })
	assertBusinessError(t, performAuthenticatedRequest(authRouter, token), bizerr.ErrInvalidToken)

	optionalRouter := gin.New()
	optionalRouter.Use(middleware.ErrorHandler(), middleware.OptionalAuth(versionedUserLookup{version: 5}, client))
	optionalRouter.GET("/private", func(c *gin.Context) {
		_, authenticated := middleware.GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"authenticated": authenticated})
	})
	response := performAuthenticatedRequest(optionalRouter, token)
	if response.Code != http.StatusOK || response.Body.String() != "{\"authenticated\":false}" {
		t.Fatalf("stale optional auth response = %d %s", response.Code, response.Body.String())
	}
}

func newAuthRouter(client redis.Cmdable) *gin.Engine {
	router := gin.New()
	router.Use(middleware.ErrorHandler(), middleware.Auth(activeUserLookup{}, client))
	router.GET("/private", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		tokenID, _ := middleware.GetTokenID(c)
		_, hasExpiry := middleware.GetTokenExpiresAt(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"token_id": tokenID,
			"has_exp":  hasExpiry,
		})
	})
	return router
}

func performAuthenticatedRequest(router http.Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertBusinessError(t *testing.T, response *httptest.ResponseRecorder, want *bizerr.Error) {
	t.Helper()
	if response.Code != want.HTTPStatus {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, want.HTTPStatus, response.Body.String())
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != want.Code {
		t.Fatalf("code = %d, want %d", body.Code, want.Code)
	}
}
