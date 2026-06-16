/**
 * 智能问答 API 封装（门户端）
 *
 * 提供会话 CRUD + SSE 流式对话接口。
 *
 * 流式对话流程：先 createSession 创建会话容器，
 * 再 streamChatMessage 在会话中发送消息并流式接收 AI 回复。
 *
 * SSE 流使用原生 fetch（非 axios），原因是 EventSource 仅支持 GET，
 * 无法携带 JSON 请求体。Token 注入和错误处理在此模块内自包含。
 *
 * TODO(chat): SSE 流绕过 Axios 拦截器——Token 刷新（401 响应）对流式请求完全失效。
 * 若 token 在流中途过期，连接断开且无恢复机制。需实现 SSE 专用的 token 刷新策略。
 */
import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'
import { getToken } from '../utils/auth'

// =============================================================================
// 类型定义
// =============================================================================

/** RAG 管道步骤事件 */
export interface StepEvent {
  id: string
  label: string
  duration_ms?: number
}

/** RAG 高级选项 */
export interface RAGOptionsParams {
  top_k?: number
  query_rewrite?: boolean
  multi_route?: boolean
  hybrid?: boolean
  rerank?: boolean
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
  /** RAG 管道执行指标（由 done 事件的 metadata 携带） */
  pipeline?: {
    steps: Array<{ id: string; label: string; duration_ms: number }>
    total_duration_ms: number
  }
}

/** 会话列表条目 */
export interface SessionListItem {
  id: number
  question: string
  last_answer: string
  message_count: number
  created_at: string
  updated_at: string
}

/** SSE 流式事件的回调签名 */
export interface StreamCallbacks {
  /** 收到文本片段时调用 */
  onToken: (content: string) => void
  /** 收到 RAG 管道步骤事件 */
  onStep?: (step: StepEvent) => void
  /** 流式传输完成，返回完整会话数据 */
  onDone: (session: ChatSessionResponse) => void
  /** 发生错误 */
  onError: (error: string) => void
}

// =============================================================================
// API 方法
// =============================================================================

/** 创建问答会话（仅创建容器，不含 AI 调用） */
export function createSession(kbId: number, title?: string) {
  return request.post<ApiResponse<{ session_id: number; kb_id: number; question: string; created_at: string }>>(
    '/api/v1/portal/chat-sessions',
    { kb_id: kbId, title }
  )
}

/**
 * 在已有会话中发送消息（SSE 流式输出）
 *
 * 使用 fetch + ReadableStream 消费 SSE 事件流，
 * 逐个 token 渲染答案，提升用户体验。
 *
 * 为什么使用 fetch 而非 EventSource：
 * EventSource 仅支持 GET 请求，无法传递 JSON 请求体，
 * 因此使用 fetch 发起 POST 并手动解析 SSE 流。
 */
export async function streamChatMessage(
  sessionId: number,
  question: string,
  callbacks: StreamCallbacks,
  signal?: AbortSignal
): Promise<void> {
  const token = getToken()

  try {
    const response = await fetch(`/api/v1/portal/chat-sessions/${sessionId}/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ question }),
      signal,
    })

    if (!response.ok) {
      const errBody = await response.json().catch(() => ({ message: '请求失败' }))
      callbacks.onError(errBody.message || `HTTP ${response.status}`)
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      callbacks.onError('浏览器不支持流式读取')
      return
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // 解析 SSE 事件（格式：data: {...}\n\n）
      const lines = buffer.split('\n\n')
      // 最后一个片段可能不完整，保留到下次处理
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const jsonStr = line.slice(6) // 去掉 "data: " 前缀

        try {
          const event = JSON.parse(jsonStr)
          if (event.type === 'token') {
            callbacks.onToken(event.content)
          } else if (event.type === 'step') {
            callbacks.onStep?.({ id: event.id, label: event.label, duration_ms: event.duration_ms })
          } else if (event.type === 'done') {
            callbacks.onDone(event.metadata as ChatSessionResponse)
          }
        } catch {
          // 跳过解析失败的 SSE 行（不完整或非 JSON）
        }
      }
    }

    // 处理 buffer 中剩余的完整事件
    if (buffer.startsWith('data: ')) {
      try {
        const event = JSON.parse(buffer.slice(6))
        if (event.type === 'done') {
          callbacks.onDone(event.metadata as ChatSessionResponse)
        }
      } catch {
        // 忽略尾部不完整数据
      }
    }
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : '网络连接失败'
    callbacks.onError(message)
  }
}

/** 获取会话列表 */
export function getSessionList(page = 1, pageSize = 10) {
  return request.get<ApiResponse<SessionListItem[]>>('/api/v1/portal/chat-sessions', {
    params: { page, page_size: pageSize },
  })
}

/** 获取会话详情（含消息历史） */
export function getChatDetail(id: number) {
  return request.get<ApiResponse<ChatSessionResponse>>(`/api/v1/portal/chat-sessions/${id}`)
}

/** 删除会话 */
export function deleteSession(id: number) {
  return request.delete<ApiResponse<null>>(`/api/v1/portal/chat-sessions/${id}`)
}

/** 提交反馈 */
export function submitFeedback(sessionID: number, feedback: number) {
  return request.post<ApiResponse<null>>(`/api/v1/portal/chat-sessions/${sessionID}/feedback`, { feedback })
}
