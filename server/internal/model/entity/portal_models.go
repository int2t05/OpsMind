// 门户问答类 GORM 实体：会话、消息、反馈
package entity

import "time"

type PortalChatSession struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	SessionNo        string     `gorm:"size:64;not null;uniqueIndex" json:"session_no"`
	UserID           *int64     `json:"user_id"`
	Question         string     `gorm:"type:text;not null" json:"question"`
	Answer           *string    `gorm:"type:text" json:"answer"`
	AnswerSource     *string    `gorm:"type:jsonb" json:"answer_source"`
	ConfidenceScore  *float64   `gorm:"type:numeric(5,2)" json:"confidence_score"`
	ModelName        *string    `gorm:"size:128" json:"model_name"`
	ModelProvider    string     `gorm:"size:64;not null;default:vllm" json:"model_provider"`
	RAGProvider      *string    `gorm:"size:64" json:"rag_provider"`
	Status           int16      `gorm:"not null;default:1" json:"status"` // 1处理中 2已完成 3转人工 4失败
	AnsweredAt       *time.Time `json:"answered_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (PortalChatSession) TableName() string { return "portal_chat_session" }

type PortalChatMessage struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	SessionID  int64     `gorm:"not null;index" json:"session_id"`
	Role       string    `gorm:"size:16;not null" json:"role"` // user/assistant/system
	Content    string    `gorm:"type:text;not null" json:"content"`
	TokenCount *int      `json:"token_count"`
	CreatedAt  time.Time `json:"created_at"`
}

func (PortalChatMessage) TableName() string { return "portal_chat_message" }

type PortalChatFeedback struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	SessionID    int64     `gorm:"not null;index" json:"session_id"`
	UserID       *int64    `json:"user_id"`
	FeedbackType int16     `gorm:"not null" json:"feedback_type"` // 1已解决 2未解决
	Remark       *string   `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
}

func (PortalChatFeedback) TableName() string { return "portal_chat_feedback" }
