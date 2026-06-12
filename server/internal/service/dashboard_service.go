// Package service 实现数据看板统计业务逻辑。
//
// DashboardService 提供看板统计和趋势数据查询功能。
//
// 为什么使用原生 SQL 而非 GORM 链式 API：
// 看板数据涉及多表聚合查询（COUNT + GROUP BY + 日期截断），
// 原生 SQL 比 GORM 链式 API 更清晰，且便于后续优化（如增加物化视图）。
//
// 为什么 AvgConfidence 只在今天范围计算：
// 置信度指标反映的是当前 AI 服务质量，历史数据参考价值有限，
// MVP 阶段仅统计今日平均值，后续可扩展为时间范围参数。
package service

import (
	"fmt"
	"time"

	"opsmind/internal/dto/request"
	"opsmind/internal/dto/response"

	"gorm.io/gorm"
)

// DashboardService 数据看板服务。
type DashboardService struct {
	db *gorm.DB
}

// NewDashboardService 创建 DashboardService 实例。
func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// =============================================================================
// GetStats
// =============================================================================

// GetStats 获取看板统计数据。
//
// 通过 SQL 聚合查询获取今日申告、各状态申告、今日问答、平均置信度和知识条目数。
// 所有统计使用 COALESCE 处理空表场景，确保返回 0 而非 NULL。
func (s *DashboardService) GetStats() (*response.StatsResponse, error) {
	var resp response.StatsResponse

	// 今日新增申告数
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM tickets WHERE created_at::date = CURRENT_DATE`,
	).Scan(&resp.TodayTickets).Error; err != nil {
		return nil, fmt.Errorf("查询今日申告数失败: %w", err)
	}

	// 待处理申告数（status=1）
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM tickets WHERE status = 1`,
	).Scan(&resp.PendingTickets).Error; err != nil {
		return nil, fmt.Errorf("查询待处理申告数失败: %w", err)
	}

	// 处理中申告数（status=2）
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM tickets WHERE status = 2`,
	).Scan(&resp.ProcessingTickets).Error; err != nil {
		return nil, fmt.Errorf("查询处理中申告数失败: %w", err)
	}

	// 已解决申告数（status=4）
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM tickets WHERE status = 4`,
	).Scan(&resp.ResolvedTickets).Error; err != nil {
		return nil, fmt.Errorf("查询已解决申告数失败: %w", err)
	}

	// 今日问答数
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM chat_sessions WHERE created_at::date = CURRENT_DATE`,
	).Scan(&resp.TodayChats).Error; err != nil {
		return nil, fmt.Errorf("查询今日问答数失败: %w", err)
	}

	// 今日平均置信度（COALESCE 确保空表返回 0）
	if err := s.db.Raw(
		`SELECT COALESCE(AVG(confidence), 0) FROM chat_sessions WHERE created_at::date = CURRENT_DATE`,
	).Scan(&resp.AvgConfidence).Error; err != nil {
		return nil, fmt.Errorf("查询平均置信度失败: %w", err)
	}

	// 知识条目总数
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM knowledge_articles`,
	).Scan(&resp.KnowledgeCount).Error; err != nil {
		return nil, fmt.Errorf("查询知识条目数失败: %w", err)
	}

	return &resp, nil
}

// =============================================================================
// GetTrends
// =============================================================================

// GetTrends 获取趋势数据。
//
// 按天聚合指定日期范围内的申告和问答数量。
// MVP 阶段仅实现 "day" 粒度，"week" 粒度降级为 "day"。
//
// 为什么先构建日期序列再 LEFT JOIN 查询：
// 纯 GROUP BY 查询不会返回零值日期，构建日期序列确保了范围内的每一天都有数据点，
// 前端可直接渲染连续曲线而无需插值处理。
func (s *DashboardService) GetTrends(req request.TrendRequest) (*response.TrendResponse, error) {
	// 解析日期范围
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("无效的开始日期: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("无效的结束日期: %w", err)
	}

	// 生成日期序列
	var dates []string
	current := startDate
	for !current.After(endDate) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}

	// 初始化结果（全部填充 0，确保零值日期也有数据点）
	dataPoints := make([]response.DataPoint, len(dates))
	for i, d := range dates {
		dataPoints[i] = response.DataPoint{
			Date:        d,
			TicketCount: 0,
			ChatCount:   0,
		}
	}

	// 查询每日申告数
	// 使用 ::date 强制转换避免时区差异导致的日期比较错误。
	// 为什么不用 DATE() 函数：DATE(timestamptz) 依赖服务器时区设置，
	// ::date 转换更明确且与索引兼容。
	type dailyCount struct {
		Date  string
		Count int64
	}

	var ticketCounts []dailyCount
	// TODO: .Scan() 错误被丢弃 — SQL 失败时静默返回空数据。
	// 应改为 if err := s.db.Raw(...).Scan(&ticketCounts).Error; err != nil { return nil, err }
	s.db.Raw(
		`SELECT TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date, COUNT(*) AS count
		 FROM tickets
		 WHERE created_at::date >= ?::date AND created_at::date <= ?::date
		 GROUP BY created_at::date
		 ORDER BY created_at::date`,
		req.StartDate, req.EndDate,
	).Scan(&ticketCounts)

	// 合并申告数据到结果
	ticketMap := make(map[string]int64, len(ticketCounts))
	for _, tc := range ticketCounts {
		ticketMap[tc.Date] = tc.Count
	}
	for i, dp := range dataPoints {
		if count, ok := ticketMap[dp.Date]; ok {
			dataPoints[i].TicketCount = count
		}
	}

	// 查询每日问答数
	var chatCounts []dailyCount
	// TODO: .Scan() 错误被丢弃 — 同上。
	s.db.Raw(
		`SELECT TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date, COUNT(*) AS count
		 FROM chat_sessions
		 WHERE created_at::date >= ?::date AND created_at::date <= ?::date
		 GROUP BY created_at::date
		 ORDER BY created_at::date`,
		req.StartDate, req.EndDate,
	).Scan(&chatCounts)

	// 合并问答数据到结果
	chatMap := make(map[string]int64, len(chatCounts))
	for _, cc := range chatCounts {
		chatMap[cc.Date] = cc.Count
	}
	for i, dp := range dataPoints {
		if count, ok := chatMap[dp.Date]; ok {
			dataPoints[i].ChatCount = count
		}
	}

	return &response.TrendResponse{DataPoints: dataPoints}, nil
}
