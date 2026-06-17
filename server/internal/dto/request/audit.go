// Package request 定义审计日志相关请求 DTO。
package request

// AuditLogListRequest 审计日志列表查询请求。
//
// OperatorID 为 0 表示不过滤操作人，Action 为空表示不过滤操作类型。
type AuditLogListRequest struct {
	OperatorID int64  `form:"operator_id"` // 操作人 ID（可选，0=全部）
	Action     string `form:"action"`      // 操作类型（可选，空=全部）
	Page       int    `form:"page"`        // 页码（默认 1）
	PageSize   int    `form:"page_size"`   // 每页条数（默认 10，最大 100）
}
