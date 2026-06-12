// Package service 实现站内消息业务逻辑。
//
// MessageService 提供消息查询、标记已读、未读计数功能，以及
// 被 TicketService 调用的 NotifySupplement 方法。
//
// 为什么 NotifySupplement 在 MessageService 而非 TicketService 中：
// 消息的格式化和写入属于消息领域的职责，TicketService 只需调用
// NotifySupplement 即可，无需关心消息内容。
package service

import (
	"errors"

	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/gorm"
)

// MessageService 站内消息服务。
type MessageService struct {
	repo *repository.MessageRepo
}

// NewMessageService 创建 MessageService 实例。
func NewMessageService(repo *repository.MessageRepo) *MessageService {
	return &MessageService{repo: repo}
}

// =============================================================================
// NotifySupplement（被 TicketService 调用）
// =============================================================================

// NotifySupplement 通知申告人补充信息。
//
// 当 TicketService 执行 request_info 操作时同步调用此方法，
// 写入一条 type=ticket_supplement 的站内消息。
// 为什么同步调用而非异步：消息写入是轻量操作（单条 INSERT），
// 同步执行可保证事务一致性——如果消息创建失败，申告操作也告失败。
func (s *MessageService) NotifySupplement(ticketID int64, userID int64) error {
	// TODO(service/message): 消息文案应包含 ticket_no/title 或跳转目标摘要。
	// 只有 RelatedID 时前端可跳转，但通知列表缺少足够上下文。
	msg := &model.Message{
		UserID:      userID,
		Title:       "申告需补充信息",
		Content:     "您的申告需要补充更多信息，请尽快登录系统查看并补充相关材料。",
		Type:        "ticket_supplement",
		RelatedType: "ticket",
		RelatedID:   ticketID,
		IsRead:      false,
	}
	return s.repo.Create(msg)
}

// =============================================================================
// 查询和操作
// =============================================================================

// ListMessages 分页查询用户消息列表。
func (s *MessageService) ListMessages(userID int64, page, pageSize int) ([]model.Message, int64, error) {
	if userID <= 0 {
		return nil, 0, AppError{Code: errcode.ErrParam, Message: "无效的用户 ID"}
	}
	return s.repo.ListByUser(userID, page, pageSize)
}

// MarkAsRead 将指定用户的消息标记为已读。
//
// 校验消息归属（userID），防止水平越权：用户 A 不能标记用户 B 的消息已读。
// 消息不存在或不属于该用户时返回 AppError{Code: ErrNotFound}。
func (s *MessageService) MarkAsRead(id int64, userID int64) error {
	if err := s.repo.MarkAsRead(id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AppError{Code: errcode.ErrNotFound, Message: "消息不存在"}
		}
		return err
	}
	return nil
}

// CountUnread 查询指定用户的未读消息数。
func (s *MessageService) CountUnread(userID int64) (int64, error) {
	// TODO(service/message): 未读数适合缓存或通过 WebSocket/SSE 推送。
	// 当前每次布局刷新都查库，用户量上来后会形成高频小查询。
	if userID <= 0 {
		return 0, AppError{Code: errcode.ErrParam, Message: "无效的用户 ID"}
	}
	return s.repo.CountUnread(userID)
}
