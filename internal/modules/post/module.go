package post

import (
    "my-bbs/internal/handler"
    "my-bbs/internal/middleware"
    "my-bbs/internal/repository"
    "my-bbs/internal/service"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

// Module 帖子模块
type Module struct {
    Handler *handler.PostHandler
}

// Initialize 初始化帖子模块
func Initialize(db *gorm.DB) *Module {
    repo := repository.NewPostRepository(db)
    svc := service.NewPostService(repo)
    hdl := handler.NewPostHandler(svc)
    return &Module{Handler: hdl}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
    // 公开路由
    r.GET("/posts", m.Handler.GetAllPosts)
    r.GET("/posts/:id", m.Handler.GetPost)

    // 需要认证的路由
    auth := r.Group("/")
    auth.Use(middleware.Auth())
    {
        auth.POST("/posts", m.Handler.CreatePost)
        auth.PUT("/posts/:id", m.Handler.UpdatePost)
        auth.DELETE("/posts/:id", m.Handler.DeletePost)
        auth.GET("/user/posts", m.Handler.GetMyPosts)
    }
}