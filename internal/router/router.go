package router

import (
	"time"

	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

// RouterDeps 路由依赖
type RouterDeps struct {
	Modules          []RouteRegister
	ReadinessChecker handler.ReadinessChecker
	HealthTimeout    time.Duration
	RateLimiter      *middleware.TieredRateLimiter
}

// SetupRouter 配置路由
func SetupRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()
	// The production reverse proxy runs on loopback and overwrites X-Real-IP.
	// Direct public clients cannot spoof this header because non-loopback peers
	// are not trusted proxy addresses.
	r.RemoteIPHeaders = []string{"X-Real-IP"}
	_ = r.SetTrustedProxies([]string{"127.0.0.1/8", "::1/128"})
	r.HandleMethodNotAllowed = true
	r.Use(
		middleware.RequestID(),
		middleware.RequestLogger(),
		middleware.Recovery(),
		middleware.ErrorHandler(),
	)
	r.NoRoute(func(c *gin.Context) {
		response.ReportError(c, bizerr.ErrNotFound)
	})
	r.NoMethod(func(c *gin.Context) {
		response.ReportError(c, bizerr.ErrMethodNotAllowed)
	})
	health := handler.NewHealthHandler(deps.ReadinessChecker, deps.HealthTimeout)
	r.GET("/livez", health.Live)
	r.GET("/readyz", middleware.LoopbackOnly(), health.Ready)

	api := r.Group("/api")
	if deps.RateLimiter != nil {
		api.Use(deps.RateLimiter.Handler())
	}

	// 每个模块自己注册路由
	for _, mod := range deps.Modules {
		mod.Register(api)
	}

	return r
}
