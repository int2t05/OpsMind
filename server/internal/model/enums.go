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
const (
	ArticleStatusDraft      int16 = 1 // 草稿
	ArticleStatusReviewing  int16 = 2 // 待审核
	ArticleStatusPublished  int16 = 3 // 已发布
	ArticleStatusDisabled   int16 = 4 // 已停用
	ArticleStatusRejected   int16 = 5 // 驳回
)

// 知识切片同步状态
const (
	ChunkSyncPending   = "pending"   // 待同步
	ChunkSyncSynced    = "synced"    // 已同步
	ChunkSyncFailed    = "failed"    // 同步失败
	ChunkSyncDisabled  = "disabled"  // 已停用
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
