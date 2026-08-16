package post

import (
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 帖子模块
type Module struct {
	Handler  *handler.PostHandler
	userRepo middleware.UserLookup
}

// Initialize 初始化帖子模块
func Initialize(db *gorm.DB) *Module {
	postRepo := gormrepo.NewPostRepository(db)
	userRepo := gormrepo.NewUserRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)
	svc := service.NewPostService(postRepo, userRepo, commentRepo, likeRepo)
	hdl := handler.NewPostHandler(svc)
	return &Module{Handler: hdl, userRepo: userRepo}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	// 公开路由（详情：公开帖所有人可读，私密帖仅作者可读；可选 Token 用于 is_liked）
	r.GET("/posts", m.Handler.GetAllPosts)
	r.GET("/posts/:id", middleware.OptionalAuth(), m.Handler.GetPost)
	r.GET("/users/:id/posts", m.Handler.GetUserPublicPosts)

	// 需要认证的路由
	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo))
	{
		auth.POST("/posts/create", m.Handler.CreatePost)
		auth.POST("/posts/update/:id", m.Handler.UpdatePost)
		auth.POST("/posts/del/:id", m.Handler.DeletePost)
		auth.POST("/posts/visible/:id", m.Handler.SetVisible)
		auth.POST("/user/posts", m.Handler.GetMyPosts)
	}
}
