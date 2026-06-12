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
 * TODO(store/chat): messages 数组中 sources 类型为 any[] — 应使用 api/chat.ts 中已定义的 SourceItem[]。
 * TODO(store/chat): pipeline_metrics 通过 (session as any) 强制类型转换获取 — 应在 ChatSessionResponse
 *                  中增加可选的 pipeline_metrics?: PipelineMetrics 字段，消除 any 断言。
 * TODO(store/chat): submitFeedback 的 catch 块静默吞错误 — 应展示用户可见的错误提示。
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
  steps: Array<{ step_id: string; label: string; duration_ms: number; success: boolean }>
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
  // State
  const currentSession = ref<ChatSessionResponse | null>(null)
  const messages = ref<Array<{ role: string; content: string; sources?: any[]; isStreaming?: boolean }>>([])
  const loading = ref(false)
  const streaming = ref(false)  // 是否正在流式输出中
  const selectedKBID = ref<number | null>(null)

  // v2: RAG 管道步骤（当前执行的步骤标签）
  const currentStep = ref('')
  // v2: 管道执行指标（done 事件时设置）
  const pipelineMetrics = ref<PipelineMetrics | null>(null)
  // v2: RAG 高级选项
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
    loading.value = true
    streaming.value = true
    selectedKBID.value = kbID
    currentStep.value = ''
    pipelineMetrics.value = null

    // 添加用户消息
    messages.value.push({ role: 'user', content: question })

    // 添加 AI 消息占位（流式填充）
    const aiMsgIndex = messages.value.length
    messages.value.push({
      role: 'assistant',
      content: '',
      sources: [],
      isStreaming: true,
    })

    await streamChatSession(
      {
        question,
        kb_id: kbID,
        rag_options: ragOptions.value,  // v2: 传递 RAG 高级选项
      },
      {
        onToken(content: string) {
          // 逐步追加 token 到 AI 消息
          const msg = messages.value[aiMsgIndex]
          if (msg) {
            msg.content += content
          }
        },
        onStep(step) {
          // v2: 更新当前管道步骤
          currentStep.value = step.label
        },
        onDone(session: ChatSessionResponse) {
          // 流式完成，更新会话元数据
          currentSession.value = session
          const msg = messages.value[aiMsgIndex]
          if (msg) {
            msg.content = session.answer
            msg.sources = session.sources
            msg.isStreaming = false
          }
          loading.value = false
          streaming.value = false
          // v2: 管道指标由 metadata 携带（如果后端支持）
          if ((session as any).pipeline_metrics) {
            pipelineMetrics.value = (session as any).pipeline_metrics
          }
        },
        onError(error: string) {
          // 流式失败时移除占位消息，显示错误
          messages.value.splice(aiMsgIndex, 1)
          messages.value.push({
            role: 'assistant',
            content: `抱歉，${error || 'AI 服务暂时不可用，请稍后重试或提交申告。'}`,
          })
          loading.value = false
          streaming.value = false
          currentStep.value = ''
        },
      }
    )
  }

  async function submitFeedback(feedback: number) {
    if (!currentSession.value) return
    try {
      await submitFeedbackApi(currentSession.value.session_id, feedback)
    } catch {
      // 静默失败
    }
  }

  function setCurrentStep(step: string) {
    currentStep.value = step
  }

  function clearSession() {
    currentSession.value = null
    messages.value = []
    currentStep.value = ''
    pipelineMetrics.value = null
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
