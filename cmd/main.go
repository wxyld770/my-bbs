package main

import (
    "log"
    "my-bbs/internal/config"
    "my-bbs/internal/database"
    "my-bbs/internal/modules/post"
    "my-bbs/internal/modules/user"
    "my-bbs/internal/router"
)

func main() {
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 初始化基础设施
    db := database.InitDB(cfg.DBDSN)

	// 3. 执行迁移
    if err := database.AutoMigrate(db); err != nil {
        log.Fatalf("迁移失败: %v", err)
    }

    // 4. 初始化各模块（每个模块封装了自己的依赖）
    userMod := user.Initialize(db)
    postMod := post.Initialize(db)

    // 5. 配置路由
    deps := router.RouterDeps{
        Modules: []router.RouteRegister{
            userMod,
            postMod,
        },
    }
    r := router.SetupRouter(deps)

    // 6. 启动服务
    addr := ":" + cfg.AppPort
    log.Printf("服务启动于 %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("启动失败: %v", err)
    }
}