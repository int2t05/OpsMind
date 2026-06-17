// Package service 实现后台调度器。
//
// Scheduler 提供定时任务管理功能，当前包含：
// - TicketAutoCloseJob：每小时检查，关闭超过 7 天的申告
//
// 为什么用 context.WithCancel 管理生命周期：
// 各 ticker goroutine 通过父 context 统一控制退出，便于优雅关闭。
//
// MessageNotifyJob 在 TicketService.UpdateStatus 中同步调用，
// 不在 Scheduler 中作为独立 goroutine 运行。
package service

import (
	"context"
	"log/slog"
	"time"
)

// ticketAutoCloseService Scheduler 需要的 TicketService 方法子集（消费者定义接口）。
type ticketAutoCloseService interface {
	AutoClose(olderThan time.Time) (int64, error)
}

// Scheduler 后台调度器。
type Scheduler struct {
	ticketSvc ticketAutoCloseService
	cancel    context.CancelFunc
}

// NewScheduler 创建 Scheduler 实例。
func NewScheduler(svc ticketAutoCloseService) *Scheduler {
	return &Scheduler{ticketSvc: svc}
}

// Start 启动调度器。
//
// 创建带取消功能的 context，启动 TicketAutoCloseJob goroutine。
// 调用 Stop() 可优雅关闭。
func (s *Scheduler) Start(ctx context.Context) {
	// TODO(service/scheduler): Start 应防止重复调用。
	// 当前重复 Start 会启动多个 auto-close goroutine，Stop 只能取消最后一次保存的 cancel。
	ctx, s.cancel = context.WithCancel(ctx)
	go s.runAutoCloseLoop(ctx)
	slog.Info("后台调度器已启动")
}

// Stop 停止调度器，取消所有 ticker goroutine。
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
		slog.Info("后台调度器已停止")
	}
}

// =============================================================================
// TicketAutoCloseJob
// =============================================================================

// runAutoCloseLoop 每小时执行一次自动关闭检查。
func (s *Scheduler) runAutoCloseLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	// TODO(service/scheduler): 启动后是否立即执行一次 AutoClose 需要明确。
	// 当前必须等 1 小时才执行，重启频繁时可能长期不清理超期申告。

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			closed, err := s.RunAutoClose(time.Now().Add(-7 * 24 * time.Hour))
			if err != nil {
				slog.Error("自动关闭申告失败", "error", err)
			} else if closed > 0 {
				slog.Info("自动关闭申告完成", "count", closed)
			}
		}
	}
}

// RunAutoClose 关闭创建时间早于 olderThan 的未完成申告。
//
// 关闭条件：status IN (1,2,3) AND created_at < olderThan。
// 返回关闭的申告数量。
// 为什么暴露为公开方法：便于测试时直接调用，无需等待 ticker。
// AutoCloseTickets 在事务中关闭申告并创建 action=auto_close 的 TicketRecord。
func (s *Scheduler) RunAutoClose(olderThan time.Time) (int64, error) {
	return s.ticketSvc.AutoClose(olderThan)
}
