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
}

// SetupRouter 配置路由
func SetupRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()
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
	r.GET("/readyz", health.Ready)

	api := r.Group("/api")

	// 每个模块自己注册路由
	for _, mod := range deps.Modules {
		mod.Register(api)
	}

	return r
}
