// Package service 实现数据看板统计业务逻辑。
//
// DashboardService 提供看板统计和趋势数据查询功能。
// 聚合查询 SQL 封装在 repository.DashboardRepo 中，遵循 3 层架构。
package service

import (
	"fmt"
	"time"

	"opsmind/internal/dto/request"
	"opsmind/internal/dto/response"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"
)

// DashboardService 数据看板服务。
type DashboardService struct {
	repo *repository.DashboardRepo
}

// NewDashboardService 创建 DashboardService 实例。
func NewDashboardService(repo *repository.DashboardRepo) *DashboardService {
	return &DashboardService{repo: repo}
}

// =============================================================================
// GetStats
// =============================================================================

// GetStats 获取看板统计数据。
func (s *DashboardService) GetStats() (*response.StatsResponse, error) {
	// TODO(service/dashboard): 这些统计查询串行执行，首屏看板延迟等于所有 SQL 延迟之和。
	// 可在 Repo 层合并为一条聚合 SQL，或在 Service 层用 errgroup 并行执行。
	var resp response.StatsResponse

	count, err := s.repo.CountTodayTickets()
	if err != nil {
		return nil, fmt.Errorf("查询今日申告数失败: %w", err)
	}
	resp.TodayTickets = count

	count, err = s.repo.CountByStatus(1) // pending
	if err != nil {
		return nil, fmt.Errorf("查询待处理申告数失败: %w", err)
	}
	resp.PendingTickets = count

	count, err = s.repo.CountByStatus(2) // processing
	if err != nil {
		return nil, fmt.Errorf("查询处理中申告数失败: %w", err)
	}
	resp.ProcessingTickets = count

	count, err = s.repo.CountByStatus(4) // resolved
	if err != nil {
		return nil, fmt.Errorf("查询已解决申告数失败: %w", err)
	}
	resp.ResolvedTickets = count

	count, err = s.repo.CountTodayChats()
	if err != nil {
		return nil, fmt.Errorf("查询今日问答数失败: %w", err)
	}
	resp.TodayChats = count

	avg, err := s.repo.AvgTodayConfidence()
	if err != nil {
		return nil, fmt.Errorf("查询平均置信度失败: %w", err)
	}
	resp.AvgConfidence = avg

	count, err = s.repo.CountKnowledgeArticles()
	if err != nil {
		return nil, fmt.Errorf("查询知识条目数失败: %w", err)
	}
	resp.KnowledgeCount = count

	return &resp, nil
}

// =============================================================================
// GetTrends
// =============================================================================

// GetTrends 获取趋势数据。
func (s *DashboardService) GetTrends(req request.TrendRequest) (*response.TrendResponse, error) {
	// TODO(service/dashboard): 校验 endDate >= startDate 且范围上限（如 90 天）。
	// 当前超大范围会生成很长日期序列并放大数据库聚合成本。
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "开始日期格式错误，格式应为 YYYY-MM-DD"}
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "结束日期格式错误，格式应为 YYYY-MM-DD"}
	}
	if endDate.Before(startDate) {
		return nil, errcode.AppError{Code: errcode.ErrParam, Message: "结束日期不能早于开始日期"}
	}

	// 生成日期序列
	var dates []string
	current := startDate
	for !current.After(endDate) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}

	// 初始化结果
	dataPoints := make([]response.DataPoint, len(dates))
	for i, d := range dates {
		dataPoints[i] = response.DataPoint{Date: d, TicketCount: 0, ChatCount: 0}
	}

	// 查询每日申告数
	// TODO(service/dashboard): 双重循环填充趋势数据是 O(days * rows)。
	// 可先把 ticketCounts/chatCounts 转为 map[date]count，再单次遍历 dataPoints。
	ticketCounts, err := s.repo.GetTicketTrends(req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("查询每日申告数失败: %w", err)
	}
	for _, tc := range ticketCounts {
		for i, dp := range dataPoints {
			if dp.Date == tc.Date {
				dataPoints[i].TicketCount = tc.Count
			}
		}
	}

	// 查询每日问答数
	chatCounts, err := s.repo.GetChatTrends(req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("查询每日问答数失败: %w", err)
	}
	for _, cc := range chatCounts {
		for i, dp := range dataPoints {
			if dp.Date == cc.Date {
				dataPoints[i].ChatCount = cc.Count
			}
		}
	}

	return &response.TrendResponse{DataPoints: dataPoints}, nil
}
