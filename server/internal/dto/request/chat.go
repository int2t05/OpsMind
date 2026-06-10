// Package request 定义智能问答相关请求 DTO。
//
// 与 TECH.md §5.2 问答 API 端点对齐。
package request

// CreateChatRequest 创建问答会话请求。
type CreateChatRequest struct {
	Question string `json:"question" binding:"required"` // 用户问题
	KBID     int64  `json:"kb_id" binding:"required"`    // 目标知识库 ID
}
