package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader is the HTTP header used to propagate a request identifier.
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	maxRequestIDLen = 128
)

type requestIDContextKey struct{}

var requestIDFallback atomic.Uint64

// RequestID assigns every request a safe identifier and returns it in the
// response header. A caller-supplied identifier is reused only when it is
// short and contains characters that cannot inject additional log lines.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(requestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		ctx := context.WithValue(c.Request.Context(), requestIDContextKey{}, requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// GetRequestID gets the identifier assigned by RequestID from a Gin context.
func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if requestID, ok := c.Get(requestIDKey); ok {
		if value, ok := requestID.(string); ok {
			return value
		}
	}
	return RequestIDFromContext(c.Request.Context())
}

// RequestIDFromContext gets the request identifier from a standard context,
// allowing application and persistence code to correlate their logs too.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func isValidRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(requestID); i++ {
		char := requestID[i]
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err == nil {
		return hex.EncodeToString(random[:])
	}

	// crypto/rand failing must not make the request itself fail. The timestamp
	// plus a process-local sequence still gives a useful correlation key.
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), requestIDFallback.Add(1))
}
