package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"my-bbs/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestRequestID_PreservesSafeCallerIDAndPropagatesContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/test", func(c *gin.Context) {
		if got := middleware.GetRequestID(c); got != "client-trace_123" {
			t.Errorf("GetRequestID() = %q", got)
		}
		if got := middleware.RequestIDFromContext(c.Request.Context()); got != "client-trace_123" {
			t.Errorf("RequestIDFromContext() = %q", got)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(middleware.RequestIDHeader, "client-trace_123")
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)

	if got := responseRecorder.Header().Get(middleware.RequestIDHeader); got != "client-trace_123" {
		t.Fatalf("response request ID = %q", got)
	}
}

func TestRequestID_ReplacesUnsafeCallerID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())
	var contextRequestID string
	router.GET("/test", func(c *gin.Context) {
		contextRequestID = middleware.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(middleware.RequestIDHeader, "unsafe request id")
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)

	got := responseRecorder.Header().Get(middleware.RequestIDHeader)
	if got == "" || got == "unsafe request id" {
		t.Fatalf("generated request ID = %q, want a non-empty replacement", got)
	}
	if contextRequestID != got {
		t.Fatalf("context request ID = %q, response header = %q", contextRequestID, got)
	}
}
