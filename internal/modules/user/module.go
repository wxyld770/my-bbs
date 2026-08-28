package user

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

// Module 用户模块
type Module struct {
	Handler  *handler.UserHandler
	userRepo middleware.UserLookup
	redis    redis.Cmdable
}

// Initialize 初始化用户模块
func Initialize(db *gorm.DB, redisClient redis.Cmdable, admins authorization.AdminChecker) *Module {
	repo := gormrepo.NewUserRepository(db)
	invitationRepo := gormrepo.NewInvitationRepository(db)
	svc := service.NewUserServiceWithRedisAndInvitations(repo, invitationRepo, redisClient, admins)
	hdl := handler.NewUserHandler(svc)
	return &Module{Handler: hdl, userRepo: repo, redis: redisClient}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	r.POST("/register", m.Handler.Register)
	r.POST("/login", m.Handler.Login)
	r.GET("/users/:id", m.Handler.GetPublicProfile)

	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo, m.redis))
	{
		auth.POST("/logout", m.Handler.Logout)
		auth.POST("/invitations", m.Handler.GenerateInvitation)
		auth.GET("/user/me", m.Handler.GetMe)
		auth.POST("/user/profile", m.Handler.UpdateProfile)
		auth.POST("/users/:id/mute", m.Handler.MuteUser)
		auth.POST("/users/:id/unmute", m.Handler.UnmuteUser)
	}
}
