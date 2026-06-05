package database

import (
	"opsmind/internal/model"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移所有数据模型。
//
// GORM AutoMigrate 会创建 GORM tag 中声明的索引，但不支持 DESC 排序。
// TECH.md §4.3 要求以下索引为 DESC：
//   - idx_tickets_created_at ON tickets(created_at DESC)
//   - idx_chat_created_at ON chat_sessions(created_at DESC)
//   - idx_audit_created_at ON audit_logs(created_at DESC)
//
// 因此 AutoMigrate 完成后，通过原始 SQL 重建这三个索引为 DESC。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.Menu{},
		&model.RoleMenu{},
		&model.Ticket{},
		&model.TicketRecord{},
		&model.KnowledgeBase{},
		&model.KnowledgeArticle{},
		&model.KnowledgeChunk{},
		&model.EmbeddingConfig{},
		&model.ChatSession{},
		&model.ChatMessage{},
		&model.AuditLog{},
		&model.SystemConfig{},
		&model.Message{},
	); err != nil {
		return err
	}

	// 重建 DESC 索引（GORM AutoMigrate 只创建 ASC 索引）
	descIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_chat_created_at ON chat_sessions(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at DESC)",
	}
	for _, sql := range descIndexes {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}
