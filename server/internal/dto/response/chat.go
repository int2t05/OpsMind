// Package response 定义智能问答相关响应 DTO。
//
// 与 TECH.md §5.2 问答 API 端点对齐。
package response

// ChatSessionResponse 问答会话响应（含答案和来源）。
type ChatSessionResponse struct {
	SessionID       int64        `json:"session_id"`
	Question        string       `json:"question"`
	Answer          string       `json:"answer"`
	Sources         []SourceItem `json:"sources"`
	Confidence      float64      `json:"confidence"`
	CanSubmitTicket bool         `json:"can_submit_ticket"`
	DurationMS      int          `json:"duration_ms"`
	Feedback        int16        `json:"feedback"`
	CreatedAt       string       `json:"created_at"`
}

// SourceItem 知识来源条目。
type SourceItem struct {
	DocName      string  `json:"doc_name"`
	ChunkContent string  `json:"chunk_content"`
	Confidence   float64 `json:"confidence"`
}
