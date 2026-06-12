/**
 * LLM 配置 API 封装（后台管理端）
 *
 * v2 新增：替代 v1 的 embedding-configs 端点，
 * 统一管理 LLM 和 Embedding 提供商配置（llama.cpp / OpenAI-compatible）。
 */
import request from '../utils/request'

// =============================================================================
// 类型定义
// =============================================================================

/** LLM 配置响应项 */
export interface LLMConfigItem {
  id: number
  name: string
  provider_type: number      // 1=llama.cpp, 2=OpenAI-compatible
  base_url: string
  embedding_base_url: string  // Embedding 独立地址，空则回退到 base_url
  api_key: string            // 返回时仅显示脱敏后的值（前4 + **** + 后4）
  llm_model: string
  embedding_model: string
  max_tokens: number
  vector_dimension: number
  is_default: boolean
  created_at: string
  updated_at: string
}

/** 创建 LLM 配置请求参数 */
export interface CreateLLMConfigParams {
  name: string
  provider_type: number
  base_url: string
  embedding_base_url?: string  // 可选，空则回退到 base_url
  api_key: string
  llm_model: string
  embedding_model: string
  max_tokens: number
  vector_dimension: number
  is_default: boolean
}

/** 更新 LLM 配置请求参数 */
export interface UpdateLLMConfigParams {
  name: string
  provider_type: number
  base_url: string
  embedding_base_url?: string
  api_key: string
  llm_model: string
  embedding_model: string
  max_tokens: number
  vector_dimension: number
  is_default: boolean
}

/** 测试连接响应 */
export interface TestConnectionResponse {
  success: boolean
  model: string
  latency: number
}

// =============================================================================
// API 方法
// =============================================================================

/** 列出全部 LLM 配置 */
export function getLLMConfigs() {
  return request.get<{ code: number; data: LLMConfigItem[] }>('/api/v1/admin/llm-configs')
}

/** 获取单个 LLM 配置详情 */
export function getLLMConfig(id: number) {
  return request.get<{ code: number; data: LLMConfigItem }>(`/api/v1/admin/llm-configs/${id}`)
}

/** 创建 LLM 配置 */
export function createLLMConfig(data: CreateLLMConfigParams) {
  return request.post<{ code: number; data: null }>('/api/v1/admin/llm-configs', data)
}

/** 更新 LLM 配置 */
export function updateLLMConfig(id: number, data: UpdateLLMConfigParams) {
  return request.put<{ code: number; data: null }>(`/api/v1/admin/llm-configs/${id}`, data)
}

/** 删除 LLM 配置 */
export function deleteLLMConfig(id: number) {
  return request.delete<{ code: number; data: null }>(`/api/v1/admin/llm-configs/${id}`)
}

/** 测试 LLM 连接 */
export function testLLMConnection(id: number) {
  return request.post<{ code: number; data: TestConnectionResponse }>(`/api/v1/admin/llm-configs/${id}/test`)
}
