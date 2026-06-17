package model

import "gorm.io/gorm"

// PaginateScope 返回 GORM 分页 Scope
// page: 页码（从 1 开始），size: 每页条数
func PaginateScope(page, size int) func(db *gorm.DB) *gorm.DB {
	// TODO(model/common): 该分页 Scope 与 repository.Paginate 重复，且没有 pageSize 上限。
	// 建议删除未使用的一份，保留统一分页入口。
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * size
		return db.Offset(offset).Limit(size)
	}
}
