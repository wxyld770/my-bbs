package user

import (
    "my-bbs/internal/handler"
    "my-bbs/internal/middleware"
    "my-bbs/internal/repository"
    "my-bbs/internal/service"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

// Module 用户模块
type Module struct {
    Handler *handler.UserHandler
}

// Initialize 初始化用户模块
func Initialize(db *gorm.DB) *Module {
    repo := repository.NewUserRepository(db)
    svc := service.NewUserService(repo)
    hdl := handler.NewUserHandler(svc)
    return &Module{Handler: hdl}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
    r.POST("/register", m.Handler.Register)
    r.POST("/login", m.Handler.Login)

    auth := r.Group("/")
    auth.Use(middleware.Auth())
    {
        auth.POST("/user/profile", m.Handler.UpdateProfile)
    }
}