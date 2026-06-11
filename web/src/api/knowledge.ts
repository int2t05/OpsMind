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
  question: string
  answer: string
  category?: string
  tags?: string[]
}

interface UpdateArticleParams {
  question: string
  answer: string
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
// Embedding 配置
// =============================================================================

interface CreateEmbeddingConfigParams {
  name: string
  model_type: number
  api_endpoint?: string
  api_key?: string
  local_path?: string
  vector_dimension: number
  is_default?: boolean
}

interface UpdateEmbeddingConfigParams {
  name: string
  model_type: number
  api_endpoint?: string
  api_key?: string
  local_path?: string
  vector_dimension: number
  is_default?: boolean
}

export function listEmbeddingConfigs() {
  return request.get('/api/v1/admin/embedding-configs')
}

export function createEmbeddingConfig(data: CreateEmbeddingConfigParams) {
  return request.post('/api/v1/admin/embedding-configs', data)
}

export function updateEmbeddingConfig(id: number, data: UpdateEmbeddingConfigParams) {
  return request.put(`/api/v1/admin/embedding-configs/${id}`, data)
}
