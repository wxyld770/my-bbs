package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-bbs/internal/config"
	"my-bbs/internal/database"
	"my-bbs/internal/logger"
	"my-bbs/internal/modules/comment"
	"my-bbs/internal/modules/like"
	"my-bbs/internal/modules/post"
	"my-bbs/internal/modules/user"
	"my-bbs/internal/router"
	"my-bbs/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// 优雅关闭超时：等待在途请求完成的最长时间
const shutdownTimeout = 10 * time.Second

func main() {
	// 1. 加载并校验配置（缺 DB_DSN / JWT_SECRET 直接退出）
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
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
	db := database.InitDB(cfg.DBDSN)

	// 4. 执行迁移
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("迁移失败: %v", err)
	}

	// 5. 初始化各模块（每个模块封装了自己的依赖）
	userMod := user.Initialize(db)
	postMod := post.Initialize(db)
	commentMod := comment.Initialize(db)
	likeMod := like.Initialize(db)

	// 6. 配置路由
	deps := router.RouterDeps{
		Modules: []router.RouteRegister{
			userMod,
			postMod,
			commentMod,
			likeMod,
		},
	}
	r := router.SetupRouter(deps)

	// 7. 启动 HTTP 服务（独立 http.Server，便于优雅关闭）
	addr := ":" + cfg.AppPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务关闭异常: %v", err)
	} else {
		logger.Info("HTTP 服务已关闭")
	}

	// 9. 释放资源：数据库 → 日志（日志在 defer 中最后关闭）
	if err := database.Close(db); err != nil {
		logger.Error("数据库关闭异常: %v", err)
	} else {
		logger.Info("数据库连接已关闭")
	}

	logger.Info("服务已优雅退出")
}
