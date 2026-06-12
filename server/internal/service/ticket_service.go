// Package service 实现申告管理业务逻辑。
//
// TicketService 提供申告 CRUD、状态机转换、处理记录管理功能。
//
// 申告状态机：待处理(1) → 处理中(2) → 需补充信息(3) → 处理中(2) → 已解决(4) / 已关闭(5)
// 为什么使用显式状态转换而非隐式条件判断：
// 状态转换规则是申告核心流程，显式状态机便于审计和调试。
package service

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"opsmind/internal/dto/request"
	"opsmind/internal/dto/response"
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TicketService 申告管理服务。
type TicketService struct {
	repo *repository.TicketRepo
}

// NewTicketService 创建 TicketService 实例。
func NewTicketService(repo *repository.TicketRepo) *TicketService {
	return &TicketService{repo: repo}
}

// =============================================================================
// CreateTicket
// =============================================================================

// CreateTicket 创建申告工单。
//
// 业务规则：
//   - title、description、contact_phone 为必填
//   - urgency 必须为 1（低）、2（中）、3（高）
//   - ticket_no 格式：TK-YYYYMMDD-XXXX（XXXX 为随机 4 位后缀）
//   - 新建申告 status=1（待处理）、source=1（门户提交）
func (s *TicketService) CreateTicket(req request.CreateTicketRequest, userID int64) error {
	// 参数校验
	if strings.TrimSpace(req.Title) == "" {
		return AppError{Code: errcode.ErrParam, Message: "标题不能为空"}
	}
	if strings.TrimSpace(req.Description) == "" {
		return AppError{Code: errcode.ErrParam, Message: "描述不能为空"}
	}
	if strings.TrimSpace(req.ContactPhone) == "" {
		return AppError{Code: errcode.ErrParam, Message: "联系电话不能为空"}
	}
	if req.Urgency < 1 || req.Urgency > 3 {
		return AppError{Code: errcode.ErrParam, Message: "紧急程度必须为 1-3"}
	}

	// 生成唯一 ticket_no
	now := time.Now()
	datePart := now.Format("20060102")
	// MVP: 使用时间戳毫秒后4位 + 随机数确保唯一
	// TODO: rand.Intn(10000) 在高并发下碰撞风险较高（仅 10000 种组合）。
	// 应改用雪花算法或数据库自增序列 + 日期前缀保证唯一性。
	suffix := fmt.Sprintf("%04d", rand.Intn(10000))
	ticketNo := fmt.Sprintf("TK-%s-%s", datePart, suffix)

	// 序列化 AffectedSystems
	var systemsJSON datatypes.JSON
	if len(req.AffectedSystems) > 0 {
		systemsJSON = marshalTicketTags(req.AffectedSystems)
	}

	// 序列化 ChatContext
	var chatCtxJSON datatypes.JSON
	if req.ChatContext != "" {
		chatCtxJSON = datatypes.JSON(req.ChatContext)
	}

	ticket := &model.Ticket{
		TicketNo:       ticketNo,
		UserID:         userID,
		Title:          req.Title,
		Description:    req.Description,
		Urgency:        int16(req.Urgency),
		ImpactScope:    int16(req.ImpactScope),
		AffectedSystems: systemsJSON,
		ContactPhone:   req.ContactPhone,
		ContactEmail:   req.ContactEmail,
		ChatContext:     chatCtxJSON,
		Status:         1,
		Source:         1,
	}

	return s.repo.Create(ticket)
}

// =============================================================================
// SupplementTicket
// =============================================================================

// SupplementTicket 补充申告信息。
//
// 业务规则：
//   - 仅申告人本人可补充
//   - 仅"需补充信息"(3)状态可补充
//   - 补充后状态变为"处理中"(2)
//   - 创建处理记录（action=supplement）
func (s *TicketService) SupplementTicket(id int64, userID int64, req request.SupplementTicketRequest) error {
	ticket, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "申告不存在"}
		}
		return err
	}

	// 仅申告人可补充
	if ticket.UserID != userID {
		return AppError{Code: errcode.ErrForbidden, Message: "仅申告人可补充信息"}
	}

	// 仅"需补充信息"状态可操作
	if ticket.Status != 3 {
		return AppError{Code: errcode.ErrParam, Message: "仅需补充信息状态可补充"}
	}

	// 创建处理记录
	record := &model.TicketRecord{
		TicketID:   id,
		OperatorID: userID,
		Action:     "supplement",
		Content:    req.Content,
	}
	if err := s.repo.CreateRecord(record); err != nil {
		return err
	}

	// 更新状态为处理中
	return s.repo.UpdateStatus(id, 2)
}

// =============================================================================
// UpdateStatus
// =============================================================================

// UpdateStatus 执行申告状态转换。
//
// 状态机规则（与 TECH.md §5.3 action 表对齐）：
//
//	start:        待处理(1) → 处理中(2)
//	request_info: 处理中(2) → 需补充信息(3)，supplement_count+1，超过3次禁止
//	resolve:      处理中(2) → 已解决(4)
//	close:        任意状态 → 已关闭(5)
//
// 每次状态转换都会创建 TicketRecord，记录操作人、操作类型和结果描述。
//
// 为什么用 switch-case 而非状态转换矩阵：
// MVP 阶段 action 数量有限（4 个），switch-case 更直观且易于调试。
func (s *TicketService) UpdateStatus(id int64, operatorID int64, req request.UpdateTicketStatusRequest) error {
	ticket, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "申告不存在"}
		}
		return err
	}

	var newStatus int16
	var recordAction string

	switch req.Action {
	case "start":
		// 仅待处理(1)可开始处理
		if ticket.Status != 1 {
			return AppError{Code: errcode.ErrParam, Message: "仅待处理状态可开始处理"}
		}
		newStatus = 2
		recordAction = "start"

	case "request_info":
		// 仅处理中(2)可请求补充信息
		if ticket.Status != 2 {
			return AppError{Code: errcode.ErrParam, Message: "仅处理中状态可请求补充信息"}
		}
		// 补充次数上限检查
		if ticket.SupplementCount >= 3 {
			return AppError{Code: errcode.ErrParam, Message: "补充信息次数已达上限（3次）"}
		}
		// 原子自增 supplement_count
		if err := s.repo.IncrementSupplementCount(id); err != nil {
			return err
		}
		newStatus = 3
		recordAction = "request_info"

	case "resolve":
		// 仅处理中(2)可解决
		if ticket.Status != 2 {
			return AppError{Code: errcode.ErrParam, Message: "仅处理中状态可解决"}
		}
		newStatus = 4
		recordAction = "resolve"

	case "close":
		// 任意状态可关闭
		newStatus = 5
		recordAction = "close"

	default:
		return AppError{Code: errcode.ErrParam, Message: "不支持的操作类型: " + req.Action}
	}

	// 更新状态
	if err := s.repo.UpdateStatus(id, int(newStatus)); err != nil {
		return err
	}

	// 创建处理记录
	record := &model.TicketRecord{
		TicketID:   id,
		OperatorID: operatorID,
		Action:     recordAction,
		Content:    req.Result,
	}
	return s.repo.CreateRecord(record)
}

// =============================================================================
// AddRecord
// =============================================================================

// AddRecord 添加处理记录（不影响状态）。
//
// 用于记录处理过程中的备注、沟通记录等。
func (s *TicketService) AddRecord(id int64, operatorID int64, req request.CreateTicketRecordRequest) error {
	// 验证申告存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "申告不存在"}
		}
		return err
	}

	var detailJSON datatypes.JSON
	if req.Detail != "" {
		detailJSON = datatypes.JSON(req.Detail)
	}

	record := &model.TicketRecord{
		TicketID:   id,
		OperatorID: operatorID,
		Action:     req.Action,
		Content:    req.Content,
		Detail:     detailJSON,
	}
	return s.repo.CreateRecord(record)
}

// =============================================================================
// ListByUser / ListAll / GetDetail
// =============================================================================

// ListByUser 分页查询当前用户的申告列表。
func (s *TicketService) ListByUser(userID int64, page, pageSize int) (*response.TicketListResponse, error) {
	tickets, total, err := s.repo.ListByUser(userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]response.TicketItem, len(tickets))
	for i, t := range tickets {
		items[i] = toTicketItem(&t)
	}

	return &response.TicketListResponse{
		Tickets: items,
		Total:   total,
	}, nil
}

// ListAll 分页查询全部申告（支持按状态和紧急程度筛选）。
//
// status=-1 表示不过滤，urgency=0 表示不过滤。
func (s *TicketService) ListAll(status, urgency, page, pageSize int) (*response.TicketListResponse, error) {
	tickets, total, err := s.repo.ListAll(status, urgency, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]response.TicketItem, len(tickets))
	for i, t := range tickets {
		items[i] = toTicketItem(&t)
	}

	return &response.TicketListResponse{
		Tickets: items,
		Total:   total,
	}, nil
}

// GetDetail 获取申告详情（含提交人信息和处理记录时间线）。
func (s *TicketService) GetDetail(id int64) (*response.TicketDetailResponse, error) {
	ticket, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AppError{Code: errcode.ErrNotFound, Message: "申告不存在"}
		}
		return nil, err
	}

	records := make([]response.TicketRecordItem, len(ticket.TicketRecords))
	for i, r := range ticket.TicketRecords {
		records[i] = response.TicketRecordItem{
			ID:         r.ID,
			TicketID:   r.TicketID,
			OperatorID: r.OperatorID,
			Action:     r.Action,
			Content:    r.Content,
			CreatedAt:  r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	detail := &response.TicketDetailResponse{
		TicketItem: toTicketItem(ticket),
	}
	detail.Description = ticket.Description
	detail.ContactEmail = ticket.ContactEmail
	detail.Source = ticket.Source
	detail.Records = records

	// 反序列化受影响的系统
	if len(ticket.AffectedSystems) > 0 {
		detail.AffectedSystems = unmarshalTicketTags(ticket.AffectedSystems)
	}

	return detail, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// toTicketItem 将 model.Ticket 转换为 TicketItem。
func toTicketItem(t *model.Ticket) response.TicketItem {
	submitterName := ""
	if t.User.ID != 0 {
		submitterName = t.User.RealName
	}

	return response.TicketItem{
		ID:              t.ID,
		TicketNo:        t.TicketNo,
		UserID:          t.UserID,
		SubmitterName:   submitterName,
		Title:           t.Title,
		Urgency:         t.Urgency,
		ImpactScope:     t.ImpactScope,
		ContactPhone:    t.ContactPhone,
		Status:          t.Status,
		StatusText:      response.TicketStatusText(t.Status),
		SupplementCount: t.SupplementCount,
		CreatedAt:       t.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// marshalTicketTags 将字符串切片序列化为 JSON。
func marshalTicketTags(items []string) datatypes.JSON {
	if len(items) == 0 {
		return datatypes.JSON(`[]`)
	}
	// 使用简单的 JSON 拼接，避免引入 encoding/json 的复杂性
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf(`"%s"`, item)
	}
	return datatypes.JSON(fmt.Sprintf(`[%s]`, strings.Join(quoted, ",")))
}

// unmarshalTicketTags 将 JSON 反序列化为字符串切片。
func unmarshalTicketTags(data datatypes.JSON) []string {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return nil
	}
	// 简单 JSON 数组解析
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
