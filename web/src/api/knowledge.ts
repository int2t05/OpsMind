/**
 * TODO(api/knowledge): 所有函数均未传入泛型类型参数，返回值全部为 any。
 *                    这是项目中 API 表面积最大的模块（17+ 端点），需要优先补全类型。
 *                    应为每个端点定义响应类型并使用 ApiResponse<T> 包装。
 * TODO(api/knowledge): listKnowledgeBases 和 listKnowledgeBasesForPortal 有独立的接口定义，
 *                     但两者结构高度相似 — 可抽取共享的 KnowledgeBaseItem 类型。
 */
import request from '../utils/request'

// =============================================================================
// 知识库 (KnowledgeBase)
// =============================================================================

interface CreateKBParams {
  name: string
  description?: string
  embedding_model?: string
}

interface UpdateKBParams {
  name: string
  description?: string
}

/** 知识库列表（后台管理用，需要 admin 权限） */
export function listKnowledgeBases() {
  return request.get('/api/v1/admin/knowledge-bases')
}

/** 知识库列表（门户端用，无需 admin 权限 — Chat 页知识库下拉框） */
export function listKnowledgeBasesForPortal() {
  return request.get('/api/v1/portal/knowledge-bases')
}

export function createKnowledgeBase(data: CreateKBParams) {
  return request.post('/api/v1/admin/knowledge-bases', data)
}

export function updateKnowledgeBase(id: number, data: UpdateKBParams) {
  return request.put(`/api/v1/admin/knowledge-bases/${id}`, data)
}

// =============================================================================
// 知识文章 (KnowledgeArticle)
// =============================================================================

interface CreateArticleParams {
  kb_id: number
  title: string
  content: string
  source_type?: number  // 1=手动, 2=上传
  category?: string
  tags?: string[]
}

interface UpdateArticleParams {
  title: string
  content: string
  category?: string
  tags?: string[]
}

interface ArticleListParams {
  page?: number
  page_size?: number
  status?: number
}

export function listArticles(kbID: number, params: ArticleListParams) {
  return request.get(`/api/v1/admin/knowledge-bases/${kbID}/articles`, { params })
}

export function getArticleDetail(id: number) {
  return request.get(`/api/v1/admin/articles/${id}`)
}

export function createArticle(kbID: number, data: CreateArticleParams) {
  return request.post(`/api/v1/admin/knowledge-bases/${kbID}/articles`, data)
}

export function updateArticle(id: number, data: UpdateArticleParams) {
  return request.put(`/api/v1/admin/articles/${id}`, data)
}

export function submitReview(id: number) {
  return request.post(`/api/v1/admin/articles/${id}/submit-review`)
}

export function reviewArticle(id: number, data: { approved: boolean; review_comment?: string }) {
  return request.post(`/api/v1/admin/articles/${id}/review`, data)
}

export function publishArticle(id: number) {
  return request.post(`/api/v1/admin/articles/${id}/publish`)
}

export function disableArticle(id: number) {
  return request.post(`/api/v1/admin/articles/${id}/disable`)
}

export function enableArticle(id: number) {
  return request.post(`/api/v1/admin/articles/${id}/enable`)
}

export function retrySyncArticle(id: number) {
  return request.post(`/api/v1/admin/articles/${id}/retry-sync`)
}

// =============================================================================
// v2 文档上传/处理（替代旧 RAG 同步）
// =============================================================================

/** 上传文档到知识库（multipart form） */
export function uploadDocuments(kbID: number, formData: FormData) {
  return request.post(`/api/v1/admin/knowledge-bases/${kbID}/documents/upload`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

/** 查询文档处理状态 */
export function getDocumentStatus(kbID: number, articleID: number) {
  return request.get(`/api/v1/admin/knowledge-bases/${kbID}/documents/${articleID}/status`)
}

/** 重试文档处理 */
export function retryDocument(kbID: number, articleID: number) {
  return request.post(`/api/v1/admin/knowledge-bases/${kbID}/documents/${articleID}/retry`)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Embedding 配置 API 已迁移至 llm_config.ts（v2 Task 6.5 清理）
//
// 旧接口 (embedding-configs) 已被后端移除，前端使用新的 LLM 配置 API 替代。
// 详见：web/src/api/llm_config.ts
// ═══════════════════════════════════════════════════════════════════════════════
