// Package repository 提供审计日志的数据访问层。
//
// AuditRepo 封装 audit_logs 表的写入和查询操作。
// 审计日志的写入由各 Service 层（用户管理、申告管理、知识管理等）调用，
// 本 Repo 同时提供服务内部的 Create 方法和面向 API 的 List 方法。
//
// 为什么没有独立的 AuditService：
// 审计日志是纯数据记录，无复杂业务逻辑（无状态机、无校验规则），
// 查询和展示直接在 Repository 层完成即可，避免过度分层。
package repository

import (
	"opsmind/internal/model"

	"gorm.io/gorm"
)

// AuditRepo 审计日志数据访问。
type AuditRepo struct {
	db *gorm.DB
}

// NewAuditRepo 创建 AuditRepo 实例。
func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Create 写入一条审计日志。
//
// 由各 Service 层在关键操作（创建/修改/删除）完成后调用。
// 写入失败会返回 error，调用方应决定是否将写入失败视为业务错误。
func (r *AuditRepo) Create(log *model.AuditLog) error {
	// TODO(repository/audit): 审计写入失败是否阻断主流程需要统一策略。
	// 关键操作（用户/角色/知识发布）通常应在同一事务中写审计，避免成功操作无审计。
	return r.db.Create(log).Error
}

// List 分页查询审计日志，支持按操作人和操作类型筛选。
//
// operatorID 为 0 时不筛选操作人，action 为空字符串时不筛选操作类型。
// 结果按 created_at DESC 排序（最新在前）。
func (r *AuditRepo) List(operatorID int64, action string, page, pageSize int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := r.db.Model(&model.AuditLog{})

	if operatorID > 0 {
		query = query.Where("operator_id = ?", operatorID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
