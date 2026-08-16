package handler

import (
	"context"
	"net/http"
	"time"

	"my-bbs/internal/logger"

	"github.com/gin-gonic/gin"
)

const defaultHealthCheckTimeout = 2 * time.Second

// ReadinessChecker 是就绪检查依赖的最小接口，*sql.DB 可直接实现。
type ReadinessChecker interface {
	PingContext(ctx context.Context) error
}

type HealthHandler struct {
	checker ReadinessChecker
	timeout time.Duration
}

func NewHealthHandler(checker ReadinessChecker, timeout time.Duration) *HealthHandler {
	if timeout <= 0 {
		timeout = defaultHealthCheckTimeout
	}
	return &HealthHandler{checker: checker, timeout: timeout}
}

// Live 表示进程仍能处理 HTTP 请求，不检查外部依赖。
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready 表示服务依赖可用；当前检查数据库连接。
func (h *HealthHandler) Ready(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	if err := h.checker.PingContext(ctx); err != nil {
		logger.Warn("数据库就绪检查失败: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
