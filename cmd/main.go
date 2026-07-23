package cmd

import (
	"log"
	"my-bbs/internal/config"
    "my-bbs/internal/database"
)

func main() {
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 初始化基础设施
    db := database.InitDB(cfg.DBDSN)

	// 3. 执行迁移（建议加开关）
    if err := database.AutoMigrate(db); err != nil {
        log.Fatalf("迁移失败: %v", err)
    }
}