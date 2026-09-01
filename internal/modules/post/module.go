package post

import (
	"my-bbs/internal/authorization"
	postcache "my-bbs/internal/cache"
	"my-bbs/internal/handler"
	"my-bbs/internal/middleware"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 帖子模块
type Module struct {
	Handler  *handler.PostHandler
	userRepo middleware.UserLookup
	redis    redis.Cmdable
}

// Initialize 初始化帖子模块
func Initialize(
	db *gorm.DB,
	redisClient redis.Cmdable,
	admins authorization.AdminChecker,
	countCaches ...*postcache.PostCountCache,
) *Module {
	var countCache *postcache.PostCountCache
	if len(countCaches) > 0 {
		countCache = countCaches[0]
	}
	postRepo := gormrepo.NewPostRepository(db)
	userRepo := gormrepo.NewUserRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)
	svc := service.NewPostServiceWithCountCache(postRepo, userRepo, commentRepo, likeRepo, countCache, admins)
	hotSvc := service.NewHotPostService(gormrepo.NewHotPostReader(db), userRepo)
	hdl := handler.NewPostHandler(svc, hotSvc)
	return &Module{Handler: hdl, userRepo: userRepo, redis: redisClient}
}

// Register 实现 router.RouteRegister 接口
func (m *Module) Register(r *gin.RouterGroup) {
	// 公开路由（详情：公开帖所有人可读，私密帖仅作者可读；可选 Token 用于 is_liked）
	r.GET("/posts/hot", m.Handler.GetHotPosts)
	r.GET("/posts", m.Handler.GetAllPosts)
	r.GET("/posts/:id", middleware.OptionalAuth(m.redis), m.Handler.GetPost)
	r.GET("/users/:id/posts", m.Handler.GetUserPublicPosts)

	// 需要认证的路由
	auth := r.Group("/")
	auth.Use(middleware.Auth(m.userRepo, m.redis))
	{
		auth.POST("/posts/create", m.Handler.CreatePost)
		auth.POST("/posts/update/:id", m.Handler.UpdatePost)
		auth.POST("/posts/del/:id", m.Handler.DeletePost)
		auth.POST("/posts/pin/:id", m.Handler.PinPost)
		auth.POST("/posts/unpin/:id", m.Handler.UnpinPost)
		auth.POST("/posts/visible/:id", m.Handler.SetVisible)
		auth.POST("/user/posts", m.Handler.GetMyPosts)
	}
}
