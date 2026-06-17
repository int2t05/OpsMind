// Package main 数据库迁移工具。
//
// 从环境变量读取数据库配置（复用 config.LoadConfig），执行 GORM AutoMigrate。
//
// 用法：
//
//	OPSMIND_DB_HOST=localhost OPSMIND_DB_PASSWORD=xxx go run ./cmd/migrate
package main

import (
	"fmt"
	"log"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/model"
)

func main() {
	// 从环境变量加载配置，开发/CI 环境自动匹配。
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	dbCfg := cfg.Database
	db, err := database.Init(dbCfg)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	if err := db.AutoMigrate(&model.Role{}, &model.UserRole{}, &model.Menu{}, &model.RoleMenu{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	fmt.Printf("Migration completed (host=%s db=%s)\n", dbCfg.Host, dbCfg.DBName)
}
