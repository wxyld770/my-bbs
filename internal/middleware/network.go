package middleware

import (
	"net"
	"strings"

	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

// LoopbackOnly hides an operator-only endpoint from non-local clients.
// SetupRouter configures Gin to accept X-Real-IP only from the loopback
// reverse proxy, so a public request proxied by nginx is still recognized as
// public while direct localhost health checks continue to work.
func LoopbackOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := net.ParseIP(strings.TrimSpace(c.ClientIP()))
		if clientIP == nil || !clientIP.IsLoopback() {
			response.ReportError(c, bizerr.ErrNotFound)
			return
		}
		c.Next()
	}
}
