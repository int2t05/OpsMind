/**
 * 问答状态管理 (Pinia)
 *
 * 管理当前问答会话状态、消息列表和加载状态。
 * 采用会话/对话分离模式：先 createSession 创建容器，
 * 再 streamChatMessage 发送消息并流式接收 AI 回复。
 *
 * 流式输出设计：
 * 先添加一条空的 assistant 消息占位，
 * 然后通过 onToken 回调逐步追加内容，实现打字机效果。
 * 流式完成后通过 onDone 回调更新完整的元数据（sources、session_id 等）。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  createSession,
  streamChatMessage,
  submitFeedback as submitFeedbackApi,
  type ChatSessionResponse,
} from '@/api/chat'

/** RAG 管道执行指标 */
export interface PipelineMetrics {
  steps: Array<{ id: string; label: string; duration_ms: number }>
  total_duration_ms: number
}

/** RAG 高级选项 */
export interface RAGOptions {
  top_k: number
  query_rewrite: boolean
  multi_route: boolean
  hybrid: boolean
  rerank: boolean
}

export const useChatStore = defineStore('chat', () => {
  // 从 sessionStorage 恢复消息（页面刷新不丢失）
  function loadMessages(): Array<{ id: string; role: string; content: string; sources?: import('@/api/chat').SourceItem[]; isStreaming?: boolean }> {
    try {
      const stored = sessionStorage.getItem('opsmind_chat_messages')
      return stored ? JSON.parse(stored) : []
    } catch { return [] }
  }
  function persistMessages(msgs: typeof messages.value) {
    try { sessionStorage.setItem('opsmind_chat_messages', JSON.stringify(msgs.slice(-50))) } catch { /* quota exceeded */ }
  }

  // State
  const currentSession = ref<ChatSessionResponse | null>(null)
  const currentSessionId = ref<number | null>(null) // 当前会话 ID（用于多轮追问）
  const messages = ref<Array<{ id: string; role: string; content: string; sources?: import('@/api/chat').SourceItem[]; isStreaming?: boolean }>>(loadMessages())
  let abortController: AbortController | null = null
  const loading = ref(false)
  const streaming = ref(false)  // 是否正在流式输出中
  const selectedKBID = ref<number | null>(null)

  // RAG 管道步骤（当前执行的步骤标签）
  const currentStep = ref('')
  // 管道执行指标（done 事件时设置）
  const pipelineMetrics = ref<PipelineMetrics | null>(null)
  // RAG 高级选项
  const ragOptions = ref<RAGOptions>({
    top_k: 5,
    query_rewrite: true,
    multi_route: true,
    hybrid: true,
    rerank: true,
  })

  // Actions

  /** 发送问题（SSE 流式模式，默认） */
  async function sendQuestion(question: string, kbID: number) {
    // 取消上一个未完成的流式请求
    if (abortController) {
      abortController.abort()
    }
    abortController = new AbortController()

    loading.value = true
    streaming.value = true
    selectedKBID.value = kbID
    currentStep.value = ''
    pipelineMetrics.value = null

    // 添加用户消息
    messages.value.push({ id: crypto.randomUUID(), role: 'user', content: question })

    // 添加 AI 消息占位（流式填充）
    const aiMsgId = crypto.randomUUID()
    messages.value.push({
      id: aiMsgId,
      role: 'assistant',
      content: '',
      sources: [],
      isStreaming: true,
    })
    persistMessages(messages.value)

    try {
      // 1. 确保会话已创建（首次对话时创建，后续复用）
      let sessionId = currentSessionId.value
      if (!sessionId) {
        const { data } = await createSession(kbID, question)
        sessionId = data.session_id
        currentSessionId.value = sessionId
      }

      // 2. 在会话中发送消息（SSE 流式）
      await streamChatMessage(
        sessionId,
        question,
        {
          onToken(content: string) {
            const msg = messages.value.find(m => m.id === aiMsgId)
            if (msg) { msg.content += content }
          },
          onStep(step) {
            currentStep.value = step.label
          },
          onDone(session: ChatSessionResponse) {
            currentSession.value = session
            const msg = messages.value.find(m => m.id === aiMsgId)
            if (msg) {
              msg.content = session.answer
              msg.sources = session.sources
              msg.isStreaming = false
            }
            loading.value = false
            streaming.value = false
            abortController = null
            persistMessages(messages.value)
            if (session.pipeline) {
              pipelineMetrics.value = session.pipeline
            }
          },
          onError(error: string) {
            const idx = messages.value.findIndex(m => m.id === aiMsgId)
            if (idx >= 0) messages.value.splice(idx, 1)
            messages.value.push({
              id: crypto.randomUUID(),
              role: 'assistant',
              content: `抱歉，${error || 'AI 服务暂时不可用，请稍后重试或提交申告。'}`,
            })
            loading.value = false
            streaming.value = false
            currentStep.value = ''
            abortController = null
            persistMessages(messages.value)
          },
        },
        abortController.signal
      )
    } catch (err: unknown) {
      // 创建会话失败时清理占位消息
      const idx = messages.value.findIndex(m => m.id === aiMsgId)
      if (idx >= 0) messages.value.splice(idx, 1)
      const message = err instanceof Error ? err.message : '网络连接失败'
      messages.value.push({
        id: crypto.randomUUID(),
        role: 'assistant',
        content: `抱歉，${message}`,
      })
      loading.value = false
      streaming.value = false
      abortController = null
      persistMessages(messages.value)
    }
  }

  async function submitFeedback(feedback: number) {
    if (!currentSession.value) return
    try {
      await submitFeedbackApi(currentSession.value.session_id, feedback)
      currentSession.value.feedback = feedback
    } catch (err) {
      console.error('提交反馈失败', err)
    }
  }

  function setCurrentStep(step: string) {
    currentStep.value = step
  }

  function clearSession() {
    if (abortController) { abortController.abort(); abortController = null }
    currentSession.value = null
    currentSessionId.value = null
    messages.value = []
    currentStep.value = ''
    pipelineMetrics.value = null
    sessionStorage.removeItem('opsmind_chat_messages')
  }

  return {
    currentSession,
    currentSessionId,
    messages,
    loading,
    streaming,
    selectedKBID,
    currentStep,
    pipelineMetrics,
    ragOptions,
    sendQuestion,
    submitFeedback,
    setCurrentStep,
    clearSession,
  }
})
