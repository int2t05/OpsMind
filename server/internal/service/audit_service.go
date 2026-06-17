// Package service 实现审计日志业务逻辑。
//
// audit_service.go 提供审计日志查询服务，统一 handler→service→repository 分层。
// AuditService 将审计日志操作统一到 Service 层，遵循 handler→service→repository 分层。
package service

import (
	"opsmind/internal/dto/response"
	"opsmind/internal/model"
	"opsmind/internal/repository"
)

// AuditService 审计日志查询服务。
// 虽然是薄封装，但维持了 Handler→Service→Repo 的分层一致性。
type AuditService struct {
	auditRepo *repository.AuditRepo
	userRepo  *repository.UserRepo
}

// NewAuditService 创建 AuditService 实例。
func NewAuditService(auditRepo *repository.AuditRepo, userRepo *repository.UserRepo) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		userRepo:  userRepo,
	}
}

// List 分页查询审计日志，附加操作人姓名。
func (s *AuditService) List(operatorID int64, action string, page, pageSize int) ([]response.AuditLogItem, int64, error) {
	// TODO(service/audit): action/target_type 应支持枚举筛选和模糊搜索。
	// 当前只能精确 action，排查“某个用户做过什么”之外的场景不够方便。
	logs, total, err := s.auditRepo.List(operatorID, action, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 收集操作人 ID 并批量查询姓名
	operatorNames := s.batchGetOperatorNames(logs)

	items := make([]response.AuditLogItem, len(logs))
	for i, log := range logs {
		detail := ""
		if len(log.Detail) > 0 {
			detail = string(log.Detail)
		}
		items[i] = response.AuditLogItem{
			ID:           log.ID,
			OperatorID:   log.OperatorID,
			OperatorName: operatorNames[log.OperatorID],
			Action:       log.Action,
			TargetType:   log.TargetType,
			TargetID:     log.TargetID,
			Detail:       detail,
			IPAddress:    log.IPAddress,
			CreatedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return items, total, nil
}

// batchGetOperatorNames 批量查询操作人姓名。
func (s *AuditService) batchGetOperatorNames(logs []model.AuditLog) map[int64]string {
	// TODO(service/audit): operatorID=0 表示系统操作，应返回“系统”而不是空字符串。
	// 这样自动关闭等日志在前端更可读。
	if len(logs) == 0 {
		return make(map[int64]string)
	}

	idSet := make(map[int64]struct{}, len(logs))
	for _, log := range logs {
		idSet[log.OperatorID] = struct{}{}
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	users, err := s.userRepo.FindByIDs(ids)
	if err != nil {
		return make(map[int64]string)
	}

	result := make(map[int64]string, len(users))
	for _, u := range users {
		result[u.ID] = u.RealName
	}
	return result
}
