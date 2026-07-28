package router

import (
    "my-bbs/internal/middleware"
    "github.com/gin-gonic/gin"
)

// RouterDeps 路由依赖
type RouterDeps struct {
    Modules []RouteRegister
}

// SetupRouter 配置路由
func SetupRouter(deps RouterDeps) *gin.Engine {
    r := gin.New()
    r.Use(
        middleware.RequestLogger(),
        middleware.Recovery(),
        middleware.ErrorHandler(),
    )

    api := r.Group("/api")

    // 每个模块自己注册路由
    for _, mod := range deps.Modules {
        mod.Register(api)
    }

    return r
}