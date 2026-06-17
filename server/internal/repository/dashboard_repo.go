// Package repository 提供数据访问层。
//
// dashboard_repo.go 定义看板聚合查询的数据访问方法。
// 聚合查询使用原生 SQL（GORM Raw），因为涉及 COUNT/GROUP BY/日期截断。
package repository

import (
	"gorm.io/gorm"
)

// DashboardRepo 看板数据访问。
type DashboardRepo struct {
	db *gorm.DB
}

// NewDashboardRepo 创建 DashboardRepo 实例。
func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

// CountTodayTickets 今日新增申告数。
func (r *DashboardRepo) CountTodayTickets() (int64, error) {
	// TODO(repository/dashboard): created_at::date 会让索引失效。
	// 建议用 created_at >= todayStart AND created_at < tomorrowStart，保持 idx_tickets_created_at 可用。
	var count int64
	err := r.db.Raw("SELECT COUNT(*) FROM tickets WHERE created_at::date = CURRENT_DATE").Scan(&count).Error
	return count, err
}

// CountByStatus 按状态统计申告数。
func (r *DashboardRepo) CountByStatus(status int16) (int64, error) {
	var count int64
	err := r.db.Raw("SELECT COUNT(*) FROM tickets WHERE status = ?", status).Scan(&count).Error
	return count, err
}

// CountTodayChats 今日问答数。
func (r *DashboardRepo) CountTodayChats() (int64, error) {
	var count int64
	err := r.db.Raw("SELECT COUNT(*) FROM chat_sessions WHERE created_at::date = CURRENT_DATE").Scan(&count).Error
	return count, err
}

// AvgTodayConfidence 今日平均置信度。
func (r *DashboardRepo) AvgTodayConfidence() (float64, error) {
	var avg float64
	err := r.db.Raw("SELECT COALESCE(AVG(confidence), 0) FROM chat_sessions WHERE created_at::date = CURRENT_DATE").Scan(&avg).Error
	return avg, err
}

// CountKnowledgeArticles 知识条目总数。
func (r *DashboardRepo) CountKnowledgeArticles() (int64, error) {
	var count int64
	err := r.db.Raw("SELECT COUNT(*) FROM knowledge_articles").Scan(&count).Error
	return count, err
}

// TrendPoint 趋势数据点（共享类型）。
type TrendPoint struct {
	Date  string
	Count int64
}

// GetTicketTrends 获取指定日期范围内的每日申告数。
func (r *DashboardRepo) GetTicketTrends(startDate, endDate string) ([]TrendPoint, error) {
	// TODO(repository/dashboard): 趋势 SQL 固定按日聚合，未支持 week granularity。
	// 如果 API 保留 granularity，应在 SQL 中按 date_trunc('week', created_at) 分支处理。
	var points []TrendPoint
	err := r.db.Raw(
		`SELECT TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date, COUNT(*) AS count
		 FROM tickets
		 WHERE created_at::date >= ?::date AND created_at::date <= ?::date
		 GROUP BY created_at::date
		 ORDER BY created_at::date`,
		startDate, endDate,
	).Scan(&points).Error
	return points, err
}

// GetChatTrends 获取指定日期范围内的每日问答数。
func (r *DashboardRepo) GetChatTrends(startDate, endDate string) ([]TrendPoint, error) {
	var points []TrendPoint
	err := r.db.Raw(
		`SELECT TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date, COUNT(*) AS count
		 FROM chat_sessions
		 WHERE created_at::date >= ?::date AND created_at::date <= ?::date
		 GROUP BY created_at::date
		 ORDER BY created_at::date`,
		startDate, endDate,
	).Scan(&points).Error
	return points, err
}
