/**
 * 问答状态管理 (Pinia)
 *
 * 管理当前问答会话状态、消息列表和加载状态。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createChatSession, submitFeedback as submitFeedbackApi, type ChatSessionResponse } from '@/api/chat'

export const useChatStore = defineStore('chat', () => {
  // State
  const currentSession = ref<ChatSessionResponse | null>(null)
  const messages = ref<Array<{ role: string; content: string; sources?: any[] }>>([])
  const loading = ref(false)
  const selectedKBID = ref<number | null>(null)

  // Actions
  async function sendQuestion(question: string, kbID: number) {
    loading.value = true
    selectedKBID.value = kbID

    // 添加用户消息到本地列表
    messages.value.push({ role: 'user', content: question })

    try {
      const res = await createChatSession({ question, kb_id: kbID })
      const data = (res as any).data || res
      currentSession.value = data

      // 添加 AI 回答到本地列表
      messages.value.push({
        role: 'assistant',
        content: data.answer,
        sources: data.sources,
      })
    } catch {
      messages.value.push({
        role: 'assistant',
        content: '抱歉，AI 服务暂时不可用，请稍后重试或提交申告。',
      })
    } finally {
      loading.value = false
    }
  }

  async function submitFeedback(feedback: number) {
    if (!currentSession.value) return
    try {
      await submitFeedbackApi(currentSession.value.session_id, feedback)
    } catch {
      // 静默失败
    }
  }

  function clearSession() {
    currentSession.value = null
    messages.value = []
  }

  return {
    currentSession,
    messages,
    loading,
    selectedKBID,
    sendQuestion,
    submitFeedback,
    clearSession,
  }
})
