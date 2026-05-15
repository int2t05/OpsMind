// 数据库迁移：基于 migration_version 表实现幂等执行
package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// ExecuteMigrations 只执行未运行过的迁移脚本，通过 migration_version 表追踪
func ExecuteMigrations(db *gorm.DB, migrationsDir string) error {
	// 确保迁移版本表存在 — 首次运行时创建
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migration_version (
			version   VARCHAR(64) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		return fmt.Errorf("create migration_version: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, f := range upFiles {
		version := strings.TrimSuffix(f, ".up.sql")

		// 检查是否已执行
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM migration_version WHERE version = ?", version).Scan(&count).Error; err != nil {
			return fmt.Errorf("check version %s: %w", version, err)
		}
		if count > 0 {
			log.Printf("migration %s already applied, skipping", f)
			continue
		}

		path := filepath.Join(migrationsDir, f)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		log.Printf("executing migration: %s", f)
		if err := db.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("execute %s: %w", f, err)
		}

		// 记录已执行
		if err := db.Exec("INSERT INTO migration_version (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("record version %s: %w", version, err)
		}
	}

	return nil
}
