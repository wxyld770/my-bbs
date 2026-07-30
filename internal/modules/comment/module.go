package comment

import (
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 评论模块
type Module struct {
	Handler *handler.CommentHandler
}

// Initialize 初始化评论模块
func Initialize(db *gorm.DB) *Module {
	commentRepo := repository.NewCommentRepository(db)
	postRepo := repository.NewPostRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewCommentService(commentRepo, postRepo, userRepo)
	hdl := handler.NewCommentHandler(svc)
	return &Module{Handler: hdl}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	r.GET("/posts/:id/comments", m.Handler.ListComments)

	auth := r.Group("/")
	auth.Use(middleware.Auth())
	{
		auth.POST("/posts/:id/comments/create", m.Handler.CreateComment)
		auth.POST("/comments/del/:id", m.Handler.DeleteComment)
	}
}
