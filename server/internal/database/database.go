// Package database 负责初始化 PostgreSQL 数据库连接。
//
// 使用 GORM 作为 ORM 框架，通过 gorm.io/driver/postgres 连接 PostgreSQL。
// RAG 向量检索由 pgvector 扩展承担，通过 adapter/pgvector_store.go 访问。
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"opsmind/internal/config"
)

// Init 初始化数据库连接。
//
// 使用 GORM 连接 PostgreSQL，配置连接池参数。
// 向量检索由 pgvector 扩展承担（通过 pgvector_store.go）。
//
// 连接池参数选择依据：
// - MaxOpenConns=25：MVP 阶段单实例部署，25 连接足够支撑并发请求
// - MaxIdleConns=10：保持空闲连接减少建连开销
// - ConnMaxLifetime=5min：避免长时间空闲连接被服务端关闭
func Init(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
