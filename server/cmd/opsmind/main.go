// OpsMind 后端服务入口：加载配置 → 初始化基础设施 → 执行迁移 → 启动 HTTP 服务
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"opsmind/server/internal/bootstrap"
	"opsmind/server/internal/router"
)

func main() {
	// 获取项目根目录，用于定位 configs/ 和 migrations/
	_, filename, _, _ := runtime.Caller(0)
	rootDir := filepath.Join(filepath.Dir(filename), "..", "..")

	configPath := filepath.Join(rootDir, "configs", "config.yaml")
	migrationsDir := filepath.Join(rootDir, "migrations")

	// 加载配置、初始化日志和数据库
	app, err := bootstrap.InitApp(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// 执行数据库迁移
	log.Println("Running database migrations...")
	if err := bootstrap.ExecuteMigrations(app.DB, migrationsDir); err != nil {
		log.Fatalf("Failed to execute migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// 设置 Gin 路由
	r := router.Setup()

	addr := fmt.Sprintf(":%d", app.Config.Server.Port)
	log.Printf("OpsMind server starting on %s (mode: %s)", addr, app.Config.Server.Mode)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
