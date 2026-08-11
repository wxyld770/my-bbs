package like

import (
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 点赞模块
type Module struct {
	Handler  *handler.LikeHandler
	userRepo *repository.UserRepository
}

// Initialize 初始化点赞模块
func Initialize(db *gorm.DB) *Module {
	likeRepo := repository.NewLikeRepository(db)
	postRepo := repository.NewPostRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewLikeService(likeRepo, postRepo)
	hdl := handler.NewLikeHandler(svc)
	return &Module{Handler: hdl, userRepo: userRepo}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo))
	{
		auth.POST("/posts/:id/like", m.Handler.ToggleLike)
	}
}
