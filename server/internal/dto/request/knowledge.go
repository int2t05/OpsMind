// Package request 定义知识库管理相关的请求结构体。
package request

// CreateKBRequest 创建知识库请求。
type CreateKBRequest struct {
	Name            string `json:"name" binding:"required"`       // 知识库名称
	Description     string `json:"description"`                    // 知识库描述
	EmbeddingModel  string `json:"embedding_model"`               // Embedding 模型名称
	VectorDimension int    `json:"vector_dimension"`              // 向量维度
	// TODO(dto/knowledge): 缺少 llm_config_id 可选字段，与 API 文档不一致。
	// Service 层 CreateKB 需处理该字段，为空时使用系统默认配置。
	LlmConfigID     int64  `json:"llm_config_id"`
}

// UpdateKBRequest 更新知识库请求。
type UpdateKBRequest struct {
	Name             string `json:"name" binding:"required"` // 知识库名称
	Description      string `json:"description"`              // 知识库描述
	EmbeddingModel   string `json:"embedding_model"`          // Embedding 模型名称（空则不更新）
	VectorDimension  int    `json:"vector_dimension"`         // 向量维度（0 则不更新）
}

// CreateArticleRequest 创建知识文章请求。
type CreateArticleRequest struct {
	KBID       int64    `json:"kb_id"`                        // 所属知识库 ID（可从路径或 JSON 获取）
	Title      string   `json:"title" binding:"required"`     // 标题（必填）
	Content    string   `json:"content" binding:"required"`   // 内容（必填）
	SourceType int16    `json:"source_type"`                  // 来源类型 1=手动 2=文档上传 3=申告转换
	Category   string   `json:"category"`                     // 分类
	Tags       []string `json:"tags"`                         // 标签列表
}

// UpdateArticleRequest 更新知识文章请求。
type UpdateArticleRequest struct {
	Title    string   `json:"title" binding:"required"`    // 标题
	Content  string   `json:"content" binding:"required"`  // 内容
	Category string   `json:"category"`                    // 分类
	Tags     []string `json:"tags"`                        // 标签列表
}

// ReviewRequest 审核知识文章请求。
type ReviewRequest struct {
	Approved      bool   `json:"approved"`       // 是否通过
	ReviewComment string `json:"review_comment"`  // 审核意见
}
