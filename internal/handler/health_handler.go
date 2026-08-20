package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/logger"
	"my-bbs/internal/middleware"

	"github.com/gin-gonic/gin"
)

const defaultHealthCheckTimeout = 2 * time.Second

// ReadinessChecker 是就绪检查依赖的最小接口，*sql.DB 可直接实现。
type ReadinessChecker interface {
	PingContext(ctx context.Context) error
}

// ReadinessCheckers combines required dependencies into one readiness check.
// Dependencies are checked with the same caller-owned deadline.
type ReadinessCheckers []ReadinessChecker

func (checkers ReadinessCheckers) PingContext(ctx context.Context) error {
	if len(checkers) == 0 {
		return fmt.Errorf("未配置就绪检查依赖")
	}
	for i, checker := range checkers {
		if checker == nil {
			return fmt.Errorf("就绪检查依赖 %d 未配置", i+1)
		}
		if err := checker.PingContext(ctx); err != nil {
			return fmt.Errorf("就绪检查依赖 %d: %w", i+1, err)
		}
	}
	return nil
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
	c.JSON(http.StatusOK, httpresp.HealthOK())
}

// Ready 表示所有必需的外部依赖均可用。
func (h *HealthHandler) Ready(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, httpresp.HealthUnavailable())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	if err := h.checker.PingContext(ctx); err != nil {
		logger.Warn("依赖就绪检查失败 | request_id=%s | error=%v", middleware.GetRequestID(c), err)
		c.JSON(http.StatusServiceUnavailable, httpresp.HealthUnavailable())
		return
	}

	c.JSON(http.StatusOK, httpresp.HealthOK())
}
