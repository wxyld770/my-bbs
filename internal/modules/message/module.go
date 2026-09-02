package message

import (
	"my-bbs/internal/authorization"
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 用户留言模块。
type Module struct {
	Handler  *handler.MessageHandler
	userRepo middleware.UserLookup
	redis    redis.Cmdable
}

func Initialize(
	db *gorm.DB,
	redisClient redis.Cmdable,
	admins authorization.AdminChecker,
) *Module {
	messageRepo := gormrepo.NewMessageRepository(db)
	userRepo := gormrepo.NewUserRepository(db)
	svc := service.NewMessageService(messageRepo, userRepo, admins)
	hdl := handler.NewMessageHandler(svc)
	return &Module{Handler: hdl, userRepo: userRepo, redis: redisClient}
}

func (m *Module) Register(r *gin.RouterGroup) {
	// 留言承担禁言申诉，因此只认证账号，不挂 RequireActiveUser。
	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo, m.redis))
	{
		auth.POST("/messages", m.Handler.CreateMessage)
		auth.GET("/messages", m.Handler.ListMyMessages)
		auth.GET("/admin/messages", m.Handler.ListAllMessages)
	}
}
