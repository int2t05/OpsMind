// Package request 定义知识库管理相关的请求结构体。
//
// 与 TECH.md §5.2 知识库管理 API 对齐。
package request

// CreateKBRequest 创建知识库请求。
type CreateKBRequest struct {
	Name            string `json:"name" binding:"required"`       // 知识库名称
	Description     string `json:"description"`                    // 知识库描述
	EmbeddingModel  string `json:"embedding_model"`               // Embedding 模型名称
	VectorDimension int    `json:"vector_dimension"`              // 向量维度
}

// UpdateKBRequest 更新知识库请求。
type UpdateKBRequest struct {
	Name        string `json:"name" binding:"required"` // 知识库名称
	Description string `json:"description"`              // 知识库描述
}

// CreateArticleRequest 创建知识文章请求（v2 title/content + 兼容 v1 question/answer）。
type CreateArticleRequest struct {
	KBID       int64    `json:"kb_id"`                        // 所属知识库 ID（可从路径或 JSON 获取）
	Question   string   `json:"question"`                     // [v1 兼容] 问题/标题
	Answer     string   `json:"answer"`                       // [v1 兼容] 答案/内容
	Title      string   `json:"title"`                        // [v2] 标题
	Content    string   `json:"content"`                      // [v2] 内容
	SourceType int16    `json:"source_type"`                  // [v2] 来源类型 1=手动 2=文档上传 3=申告转换
	Category   string   `json:"category"`                     // 分类
	Tags       []string `json:"tags"`                         // 标签列表
}

// UpdateArticleRequest 更新知识文章请求。
type UpdateArticleRequest struct {
	// TODO(dto/knowledge): 更新文章同样需要迁移到 title/content，并支持 tags/category 的局部更新语义。
	Question string   `json:"question" binding:"required"` // 问题
	Answer   string   `json:"answer" binding:"required"`   // 答案
	Category string   `json:"category"`                    // 分类
	Tags     []string `json:"tags"`                       // 标签列表
}

// ReviewRequest 审核知识文章请求。
type ReviewRequest struct {
	Approved      bool   `json:"approved"`       // 是否通过
	ReviewComment string `json:"review_comment"`  // 审核意见
}
