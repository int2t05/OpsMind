// Package repository 实现数据访问层。
//
// pagination.go 提供通用分页辅助函数，消除各 Repo List 方法中的重复分页逻辑。
//
// TODO: 所有 Repository 方法缺少 context.Context 参数——DB 查询无法感知 HTTP 超时/取消，
// tracing span 无法传播到 SQL 层。应改为 Method(ctx context.Context, ...) + r.db.WithContext(ctx)。
package repository

import "gorm.io/gorm"

// Paginate 执行通用分页查询。
//
// 为什么用泛型而非每 repo 单独实现：
// 7 个 repo 的 List 方法有完全相同的 Count + Offset + Limit + Order 模式，
// 泛型函数将分页逻辑集中一处，各 repo 只需一行调用。
//
// TODO: 此函数当前零调用方——7 个 Repo 仍在手动实现分页（约 60 行重复代码）。
// 应统一迁移到此泛型函数，或如团队偏好显式代码则删除此文件避免误导。
//
// T 为模型类型（如 model.KnowledgeArticle），返回查询结果切片、总数和可能的错误。
// order 为排序子句（如 "id DESC"），空字符串表示不排序。
func Paginate[T any](query *gorm.DB, page, pageSize int, order string) ([]T, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if order != "" {
		query = query.Order(order)
	}

	var results []T
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}
