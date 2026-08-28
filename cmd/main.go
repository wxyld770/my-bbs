package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"my-bbs/internal/authorization"
	"my-bbs/internal/config"
	"my-bbs/internal/database"
	"my-bbs/internal/handler"
	"my-bbs/internal/logger"
	"my-bbs/internal/middleware"
	"my-bbs/internal/modules/comment"
	"my-bbs/internal/modules/like"
	"my-bbs/internal/modules/post"
	"my-bbs/internal/modules/search"
	"my-bbs/internal/modules/user"
	"my-bbs/internal/redisstore"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/router"
	"my-bbs/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载并校验配置（缺 DB_DSN / JWT_SECRET / REDIS_ADDR / ADMIN_USERNAMES 直接退出）
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	adminUsers, err := authorization.ParseAdminUsers(cfg.AdminUsernames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "管理员配置错误: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志（写入当前目录下的日志文件夹）
	if err := logger.Init(cfg.LogDir); err != nil {
		logger.Fatal("日志初始化失败: %v", err)
	}
	defer logger.Close()

	jwt.Init(cfg.JWTSecret)
	gin.SetMode(cfg.AppMode)

	// 3. 初始化基础设施
	db, sqlDB, err := database.InitDB(cfg.DBDSN, database.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		logger.Fatal("数据库初始化失败: %v", err)
	}
	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.HealthCheckTimeout)
	if err := sqlDB.PingContext(startupCtx); err != nil {
		startupCancel()
		logger.Fatal("数据库连通性检查失败: %v", err)
	}
	startupCancel()

	redisStartupCtx, redisStartupCancel := context.WithTimeout(context.Background(), cfg.HealthCheckTimeout)
	redisClient, err := redisstore.Open(redisStartupCtx, redisstore.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})
	redisStartupCancel()
	if err != nil {
		logger.Fatal("Redis 初始化失败: %v", err)
	}

	// 4. 开发环境可自动迁移；生产环境由发布流程执行版本内的 full_ddl.sql。
	if cfg.DBAutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			logger.Fatal("迁移失败: %v", err)
		}
	} else {
		logger.Info("数据库自动迁移已关闭，使用部署流程管理数据库结构")
	}
	adminCheckCtx, adminCheckCancel := context.WithTimeout(context.Background(), cfg.HealthCheckTimeout)
	if err := authorization.ValidateExistingAdminUsers(
		adminCheckCtx,
		gormrepo.NewUserRepository(db),
		adminUsers,
	); err != nil {
		adminCheckCancel()
		logger.Fatal("管理员账号配置校验失败: %v", err)
	}
	adminCheckCancel()

	// 5. 初始化各模块（每个模块封装了自己的依赖）
	userMod := user.Initialize(db, redisClient, adminUsers)
	postMod := post.Initialize(db, redisClient, adminUsers)
	commentMod := comment.Initialize(db, redisClient)
	likeMod := like.Initialize(db, redisClient)
	searchMod := search.Initialize(db, cfg.SearchTimeout)
	rateLimiter, err := middleware.NewTieredRateLimiter(middleware.TieredRateLimitConfig{
		Login: middleware.RateLimitPolicy{
			Requests: cfg.RateLimitLoginRequests,
			Window:   cfg.RateLimitLoginWindow,
			Burst:    cfg.RateLimitLoginBurst,
		},
		Register: middleware.RateLimitPolicy{
			Requests: cfg.RateLimitRegisterRequests,
			Window:   cfg.RateLimitRegisterWindow,
			Burst:    cfg.RateLimitRegisterBurst,
		},
		Search: middleware.RateLimitPolicy{
			Requests: cfg.RateLimitSearchRequests,
			Window:   cfg.RateLimitSearchWindow,
			Burst:    cfg.RateLimitSearchBurst,
		},
		Write: middleware.RateLimitPolicy{
			Requests: cfg.RateLimitWriteRequests,
			Window:   cfg.RateLimitWriteWindow,
			Burst:    cfg.RateLimitWriteBurst,
		},
		Read: middleware.RateLimitPolicy{
			Requests: cfg.RateLimitReadRequests,
			Window:   cfg.RateLimitReadWindow,
			Burst:    cfg.RateLimitReadBurst,
		},
		MaxEntries: cfg.RateLimitMaxEntries,
		IdleTTL:    cfg.RateLimitIdleTTL,
	})
	if err != nil {
		logger.Fatal("限流器初始化失败: %v", err)
	}

	// 6. 配置路由
	deps := router.RouterDeps{
		Modules: []router.RouteRegister{
			userMod,
			postMod,
			commentMod,
			likeMod,
			searchMod,
		},
		ReadinessChecker: handler.ReadinessCheckers{sqlDB, redisClient},
		HealthTimeout:    cfg.HealthCheckTimeout,
		RateLimiter:      rateLimiter,
	}
	r := router.SetupRouter(deps)

	// 7. 启动 HTTP 服务（独立 http.Server，便于优雅关闭）
	addr := ":" + cfg.AppPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	// 监听 SIGINT / SIGTERM，收到后取消 ctx
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("服务启动于 %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("启动失败: %v", err)
		}
	}()

	// 阻塞直到收到退出信号
	<-ctx.Done()
	stop()
	logger.Info("收到退出信号，开始优雅关闭...")

	// 8. 优雅关闭：停止接新请求，等待在途请求结束（或超时）
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务关闭异常: %v", err)
	} else {
		logger.Info("HTTP 服务已关闭")
	}

	// 9. 释放资源：Redis → 数据库 → 日志（日志在 defer 中最后关闭）
	if err := redisClient.Close(); err != nil {
		logger.Error("Redis 连接关闭异常: %v", err)
	} else {
		logger.Info("Redis 连接已关闭")
	}

	if err := database.Close(sqlDB); err != nil {
		logger.Error("数据库关闭异常: %v", err)
	} else {
		logger.Info("数据库连接已关闭")
	}

	logger.Info("服务已优雅退出")
}
