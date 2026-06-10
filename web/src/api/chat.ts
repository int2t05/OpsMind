/**
 * 智能问答 API 封装（门户端）
 *
 * 提供问答会话创建和反馈提交接口。
 */
import request from '../utils/request'

// =============================================================================
// 类型定义
// =============================================================================

export interface CreateChatParams {
  question: string
  kb_id: number
}

export interface SourceItem {
  doc_name: string
  chunk_content: string
  confidence: number
}

export interface ChatSessionResponse {
  session_id: number
  question: string
  answer: string
  sources: SourceItem[]
  confidence: number
  can_submit_ticket: boolean
  duration_ms: number
  feedback: number
  created_at: string
}

// =============================================================================
// API 方法
// =============================================================================

/** 创建问答会话 */
export function createChatSession(data: CreateChatParams) {
  return request.post<ChatSessionResponse>('/api/v1/portal/chat-sessions', data)
}

/** 提交反馈 */
export function submitFeedback(sessionID: number, feedback: number) {
  return request.post(`/api/v1/portal/chat-sessions/${sessionID}/feedback`, { feedback })
}
