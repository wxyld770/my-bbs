package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"my-bbs/internal/middleware"
	jwtpkg "my-bbs/pkg/jwt"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestAuthPublishesVerifiedSessionVersionForPasswordCAS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("password-cas-session-version-context-secret")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	const sessionVersion uint64 = 9
	token, err := jwtpkg.GenerateTokenWithSessionVersion(77, sessionVersion)
	if err != nil {
		t.Fatalf("GenerateTokenWithSessionVersion() error = %v", err)
	}

	router := gin.New()
	router.Use(
		middleware.ErrorHandler(),
		middleware.Auth(versionedUserLookup{version: sessionVersion}, client),
	)
	router.POST("/password", func(c *gin.Context) {
		version, ok := middleware.GetSessionVersion(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"session_version": version})
	})

	req := httptest.NewRequest(http.MethodPost, "/password", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK || response.Body.String() != "{\"session_version\":9}" {
		t.Fatalf("verified session-version context response=%d %s", response.Code, response.Body.String())
	}
}

func TestGetSessionVersionRejectsMissingContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/missing", func(c *gin.Context) {
		if _, ok := middleware.GetSessionVersion(c); ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("missing context status=%d, want=%d", response.Code, http.StatusNoContent)
	}
}
