package model

// 用户状态
const (
	StatusActive   int16 = 1 // 正常
	StatusInactive int16 = 2 // 冻结
)

// 工单紧急程度
const (
	TicketUrgencyLow    int16 = 1 // 低
	TicketUrgencyMedium int16 = 2 // 中
	TicketUrgencyHigh   int16 = 3 // 高
)

// 工单影响范围
const (
	ImpactPersonal int16 = 1 // 个人
	ImpactDept     int16 = 2 // 部门
	ImpactCompany  int16 = 3 // 全公司
)

// 工单状态
const (
	TicketStatusPending         int16 = 1 // 待处理
	TicketStatusProcessing      int16 = 2 // 处理中
	TicketStatusNeedSupplement  int16 = 3 // 需补充信息
	TicketStatusResolved        int16 = 4 // 已解决
	TicketStatusClosed          int16 = 5 // 已关闭
)

// 工单来源
const (
	TicketSourcePortal int16 = 1 // 门户提交
	TicketSourceChat   int16 = 2 // 问答转申告
)

// 工单操作类型
const (
	TicketActionStart        = "start"         // 开始处理
	TicketActionRequestInfo  = "request_info"  // 要求补充信息
	TicketActionSupplement   = "supplement"     // 补充信息
	TicketActionResolve      = "resolve"        // 解决
	TicketActionClose        = "close"          // 关闭
)

// 知识文章状态
//
// TODO(model/enums): 状态编号与 docs/API/knowledge.md 不一致——文档定义「已停用(5)」，
// 代码中 Disabled=0。API 文档的生命周期图缺少 Disabled=0 状态的完整描述。
// 需统一编号方案并更新文档。
const (
	ArticleStatusDisabled   int16 = 0 // 已停用
	ArticleStatusDraft      int16 = 1 // 草稿
	ArticleStatusReviewing  int16 = 2 // 待审核
	ArticleStatusApproved   int16 = 3 // 审核通过
	ArticleStatusPublished  int16 = 4 // 已发布
	ArticleStatusRejected   int16 = 5 // 驳回
)


// Embedding 模型类型
const (
	EmbeddingTypeAPI   int16 = 1 // API 接入
	EmbeddingTypeLocal int16 = 2 // 本地部署
)

// 对话角色
const (
	ChatRoleUser      = "user"
	ChatRoleAssistant  = "assistant"
)

// 问答反馈状态
const (
	ChatFeedbackUnset     int16 = 0 // 未反馈
	ChatFeedbackResolved  int16 = 1 // 已解决
	ChatFeedbackUnresolved int16 = 2 // 未解决
)

// 站内消息类型
const (
	MessageTypeTicketSupplement = "ticket_supplement" // 申告补充信息
	MessageTypeSystem           = "system"            // 系统通知
)

// 菜单类型
const (
	MenuTypeMenu    = "menu"    // 菜单
	MenuTypeButton  = "button"  // 按钮
)

// TicketStatusText 返回工单状态的中文描述。
//
// 为什么放在 model 包而非 DTO：业务映射函数与状态常量就近维护，
// 避免 DTO 包承担数据模型之外的职责。
func TicketStatusText(status int16) string {
	// TODO(model/enums): 为知识文章、处理状态、紧急程度、影响范围也提供统一 Text 方法。
	// 当前这些映射散落在 Service 和前端工具函数中，容易出现文案不一致。
	switch status {
	case TicketStatusPending:
		return "待处理"
	case TicketStatusProcessing:
		return "处理中"
	case TicketStatusNeedSupplement:
		return "需补充信息"
	case TicketStatusResolved:
		return "已解决"
	case TicketStatusClosed:
		return "已关闭"
	default:
		return "未知"
	}
}
