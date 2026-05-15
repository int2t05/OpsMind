// 基础仓库：封装 GORM 事务、分页查询
package repository

import (
	"math"

	"gorm.io/gorm"
)

// Paginate 对 GORM 查询应用分页，返回总条数
func Paginate(db *gorm.DB, page, perPage int) (*gorm.DB, int64) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var total int64
	db.Count(&total)

	offset := (page - 1) * perPage
	return db.Offset(offset).Limit(perPage), total
}

// TotalPages 计算总页数
func TotalPages(total int64, perPage int) int64 {
	return int64(math.Ceil(float64(total) / float64(perPage)))
}
