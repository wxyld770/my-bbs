package middleware_test

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"my-bbs/internal/middleware"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestErrorHandler_LogsCauseOnceAndReturnsSafeResponse(t *testing.T) {
	logs := captureStandardLogs(t)
	router := testErrorRouter()
	router.GET("/failure", func(c *gin.Context) {
		response.ReportError(c, errors.New("database-password-secret"))
	})

	request := httptest.NewRequest(http.MethodGet, "/failure?token=query-secret", nil)
	request.Header.Set(middleware.RequestIDHeader, "trace-123")
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body=%s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if strings.Contains(responseRecorder.Body.String(), "database-password-secret") {
		t.Fatalf("internal cause leaked in response: %s", responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), bizerr.ErrInternal.Message) {
		t.Fatalf("safe internal response missing: %s", responseRecorder.Body.String())
	}
	if got := responseRecorder.Header().Get(middleware.RequestIDHeader); got != "trace-123" {
		t.Fatalf("response request ID = %q", got)
	}

	logOutput := logs.String()
	if count := strings.Count(logOutput, "database-password-secret"); count != 1 {
		t.Fatalf("cause log count = %d, logs=%s", count, logOutput)
	}
	if !strings.Contains(logOutput, "request_id=trace-123") {
		t.Fatalf("request ID missing from logs: %s", logOutput)
	}
	if strings.Contains(logOutput, "query-secret") {
		t.Fatalf("raw query leaked into logs: %s", logOutput)
	}
}

func TestRequestLogger_DoesNotLogErrorsOrRawQuery(t *testing.T) {
	logs := captureStandardLogs(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.RequestLogger())
	router.GET("/items", func(c *gin.Context) {
		_ = c.Error(errors.New("private-error-marker"))
		c.Status(http.StatusTeapot)
	})

	request := httptest.NewRequest(http.MethodGet, "/items?token=query-secret", nil)
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)

	logOutput := logs.String()
	if strings.Contains(logOutput, "private-error-marker") {
		t.Fatalf("request logger logged c.Errors: %s", logOutput)
	}
	if strings.Contains(logOutput, "query-secret") || strings.Contains(logOutput, "?token=") {
		t.Fatalf("request logger logged raw query: %s", logOutput)
	}
	for _, want := range []string{"method=GET", "route=/items", "path=/items", "status=418", "request_id="} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("request log missing %q: %s", want, logOutput)
		}
	}
}

func TestRecovery_DirectlyWritesSafe500WithoutDumpingHeadersOrQuery(t *testing.T) {
	logs := captureStandardLogs(t)
	router := testErrorRouter()
	router.GET("/panic", func(c *gin.Context) {
		panic("panic-cause")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic?token=query-secret", nil)
	request.Header.Set(middleware.RequestIDHeader, "panic-trace")
	request.Header.Set("Authorization", "Bearer authorization-secret")
	request.Header.Set("X-API-Key", "header-secret")
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body=%s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), bizerr.ErrInternal.Message) {
		t.Fatalf("safe internal response missing: %s", responseRecorder.Body.String())
	}
	if strings.Contains(responseRecorder.Body.String(), "panic-cause") {
		t.Fatalf("panic leaked in response: %s", responseRecorder.Body.String())
	}

	logOutput := logs.String()
	for _, secret := range []string{"authorization-secret", "header-secret", "query-secret"} {
		if strings.Contains(logOutput, secret) {
			t.Fatalf("secret %q leaked into recovery logs: %s", secret, logOutput)
		}
	}
	if !strings.Contains(logOutput, "panic-cause") || !strings.Contains(logOutput, "request_id=panic-trace") {
		t.Fatalf("panic cause or request ID missing from logs: %s", logOutput)
	}
}

func testErrorRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.RequestLogger(), middleware.Recovery(), middleware.ErrorHandler())
	return router
}

func captureStandardLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	logs := &bytes.Buffer{}
	log.SetOutput(logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	return logs
}
