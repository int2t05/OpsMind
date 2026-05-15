// 应用装配：初始化 DB、缓存、日志、路由和外部适配器
package bootstrap

import (
	"fmt"
	"log"

	"opsmind/server/internal/config"
	"opsmind/server/internal/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// App 持有所有基础设施依赖
type App struct {
	Config *config.Config
	DB     *gorm.DB
}

// InitApp 初始化应用：加载配置 → 日志 → 数据库 → 返回 App
func InitApp(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}

	if err := logger.Init(cfg.Log.Level, cfg.Log.Output); err != nil {
		return nil, fmt.Errorf("logger.Init: %w", err)
	}

	logLevel := gormlogger.Silent
	if cfg.Log.Level == "debug" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm.Open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	log.Println("database connected successfully")

	return &App{Config: cfg, DB: db}, nil
}
