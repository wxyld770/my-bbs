package comment

import (
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 评论模块
type Module struct {
	Handler  *handler.CommentHandler
	userRepo middleware.UserLookup
	redis    redis.Cmdable
}

// Initialize 初始化评论模块
func Initialize(db *gorm.DB, redisClient redis.Cmdable) *Module {
	commentRepo := gormrepo.NewCommentRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	userRepo := gormrepo.NewUserRepository(db)
	svc := service.NewCommentService(commentRepo, postRepo, userRepo)
	hdl := handler.NewCommentHandler(svc)
	return &Module{Handler: hdl, userRepo: userRepo, redis: redisClient}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	r.GET("/posts/:id/comments", m.Handler.ListComments)

	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo, m.redis))
	{
		auth.POST("/posts/:id/comments/create", m.Handler.CreateComment)
		auth.POST("/comments/del/:id", m.Handler.DeleteComment)
	}
}
