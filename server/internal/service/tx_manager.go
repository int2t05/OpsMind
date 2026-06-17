package service

import "gorm.io/gorm"

// TxManager 事务管理器接口。
//
// 为跨 Repository 的事务操作提供统一抽象，避免 Service 直接持有 *gorm.DB。
type TxManager interface {
	Transaction(fn func(tx *gorm.DB) error) error
}

// GormTxManager 基于 GORM 的 TxManager 实现。
type GormTxManager struct {
	db *gorm.DB
}

func NewGormTxManager(db *gorm.DB) *GormTxManager {
	// TODO(service/tx): 校验 db 非 nil，构造期提前暴露装配错误。
	// 现在 nil db 会在第一次 Transaction 时 panic。
	return &GormTxManager{db: db}
}

func (m *GormTxManager) Transaction(fn func(tx *gorm.DB) error) error {
	// TODO(service/tx): Transaction 可以接收 context.Context 并使用 db.WithContext(ctx)。
	// 这样事务内 SQL 也能响应请求取消和超时。
	return m.db.Transaction(fn)
}
