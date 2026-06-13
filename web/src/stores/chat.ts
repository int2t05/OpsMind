/**
 * 问答状态管理 (Pinia)
 *
 * 管理当前问答会话状态、消息列表和加载状态。
 * 支持普通模式和 SSE 流式输出两种问答方式。
 *
 * 流式输出设计：
 * 流式模式下，先添加一条空的 assistant 消息占位，
 * 然后通过 onToken 回调逐步追加内容，实现打字机效果。
 * 流式完成后通过 onDone 回调更新完整的元数据（sources、session_id 等）。
 *
 * submitFeedback 的 catch 块已添加 console.error。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  streamChatSession,
  submitFeedback as submitFeedbackApi,
  type ChatSessionResponse,
} from '@/api/chat'

/** RAG 管道执行指标 */
export interface PipelineMetrics {
  steps: Array<{ id: string; label: string; duration_ms: number; success: boolean }>
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
  const messages = ref<Array<{ id: string; role: string; content: string; sources?: import('@/api/chat').SourceItem[]; isStreaming?: boolean }>>(loadMessages())
  let abortController: AbortController | null = null
  const loading = ref(false)
  const streaming = ref(false)  // 是否正在流式输出中
  const selectedKBID = ref<number | null>(null)

  //RAG 管道步骤（当前执行的步骤标签）
  const currentStep = ref('')
  //  管道执行指标（done 事件时设置）
  const pipelineMetrics = ref<PipelineMetrics | null>(null)
  //RAG 高级选项
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

    await streamChatSession(
      {
        question,
        kb_id: kbID,
        rag_options: ragOptions.value,
      },
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
  }

  async function submitFeedback(feedback: number) {
    if (!currentSession.value) return
    try {
      await submitFeedbackApi(currentSession.value.session_id, feedback)
      currentSession.value.feedback = feedback
    } catch (err) {
      // TODO(web/stores/chat): 反馈提交失败仅 console.error，用户点击后静默失败。
      // 应通过 toast 提示用户重试，避免用户误以为反馈已生效。
      console.error('提交反馈失败', err)
    }
  }

  function setCurrentStep(step: string) {
    currentStep.value = step
  }

  function clearSession() {
    if (abortController) { abortController.abort(); abortController = null }
    currentSession.value = null
    messages.value = []
    currentStep.value = ''
    pipelineMetrics.value = null
    sessionStorage.removeItem('opsmind_chat_messages')
  }

  return {
    currentSession,
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
