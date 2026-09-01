package like

import (
	postcache "my-bbs/internal/cache"
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 点赞模块
type Module struct {
	Handler  *handler.LikeHandler
	userRepo middleware.UserLookup
	redis    redis.Cmdable
}

// Initialize 初始化点赞模块
func Initialize(db *gorm.DB, redisClient redis.Cmdable, countCaches ...*postcache.PostCountCache) *Module {
	var countCache *postcache.PostCountCache
	if len(countCaches) > 0 {
		countCache = countCaches[0]
	}
	likeRepo := gormrepo.NewLikeRepository(db)
	postRepo := gormrepo.NewPostRepository(db)
	userRepo := gormrepo.NewUserRepository(db)
	svc := service.NewLikeServiceWithCountCache(likeRepo, postRepo, userRepo, countCache)
	hdl := handler.NewLikeHandler(svc)
	return &Module{Handler: hdl, userRepo: userRepo, redis: redisClient}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo, m.redis))
	auth.Use(middleware.RequireActiveUser())
	{
		auth.POST("/posts/:id/like", m.Handler.ToggleLike)
	}
}
