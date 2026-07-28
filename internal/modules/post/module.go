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
    postRepo := repository.NewPostRepository(db)
    userRepo := repository.NewUserRepository(db)
    svc := service.NewPostService(postRepo, userRepo)
    hdl := handler.NewPostHandler(svc)
    return &Module{Handler: hdl}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
    // 公开路由（详情仅返回公开帖）
    r.GET("/posts", m.Handler.GetAllPosts)
    r.GET("/posts/:id", m.Handler.GetPost)

    // 需要认证的路由
    auth := r.Group("/")
    auth.Use(middleware.Auth())
    {
        auth.POST("/posts/create", m.Handler.CreatePost)
        auth.POST("/posts/update/:id", m.Handler.UpdatePost)
        auth.POST("/posts/del/:id", m.Handler.DeletePost)
        auth.POST("/posts/visible/:id", m.Handler.SetVisible)
        auth.POST("/user/posts", m.Handler.GetMyPosts)
    }
}