// 申告单类 GORM 实体：工单、处理记录、回访记录
package entity

import "time"

type Ticket struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	TicketNo         string    `gorm:"size:64;not null;uniqueIndex" json:"ticket_no"`
	SourceSessionID  *int64    `json:"source_session_id"`
	SourceFeedbackID *int64    `json:"source_feedback_id"`
	ReporterName     string    `gorm:"size:64;not null" json:"reporter_name"`
	ReporterPhone    string    `gorm:"size:32;not null" json:"reporter_phone"`
	Title            string    `gorm:"size:255;not null" json:"title"`
	Description      string    `gorm:"type:text;not null" json:"description"`
	ImpactScope      *string   `gorm:"size:255" json:"impact_scope"`
	UrgencyLevel     int16     `gorm:"not null" json:"urgency_level"` // 1低 2中 3高
	Status           int16     `gorm:"not null;default:1" json:"status"` // 1待处理 2处理中 3待补充 4已完成 5已关闭
	AssigneeID       *int64    `json:"assignee_id"`
	AIContext        *string   `gorm:"type:jsonb" json:"ai_context"`
	AttachmentCount  int       `gorm:"not null;default:0" json:"attachment_count"`
	ClosedReason     *string   `gorm:"size:255" json:"closed_reason"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Ticket) TableName() string { return "ticket" }

type TicketProcessRecord struct {
	ID                int64     `gorm:"primaryKey" json:"id"`
	TicketID          int64     `gorm:"not null;index" json:"ticket_id"`
	HandlerID         int64     `gorm:"not null" json:"handler_id"`
	ProcessStatus     int16     `gorm:"not null" json:"process_status"`
	ProcessContent    string    `gorm:"type:text;not null" json:"process_content"`
	ProcessResult     *string   `gorm:"type:text" json:"process_result"`
	RequiresMoreInfo  bool      `gorm:"not null;default:false" json:"requires_more_info"`
	CreatedAt         time.Time `json:"created_at"`
}

func (TicketProcessRecord) TableName() string { return "ticket_process_record" }

type TicketVisitRecord struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	TicketID     int64     `gorm:"not null;index" json:"ticket_id"`
	VisitorID    int64     `gorm:"not null" json:"visitor_id"`
	VisitResult  int16     `gorm:"not null" json:"visit_result"` // 1满意 2一般 3不满意
	VisitContent *string   `gorm:"size:500" json:"visit_content"`
	CreatedAt    time.Time `json:"created_at"`
}

func (TicketVisitRecord) TableName() string { return "ticket_visit_record" }
