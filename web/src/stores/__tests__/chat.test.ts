import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChatStore } from '../chat'

// Mock API 模块
vi.mock('../../api/chat', () => ({
  createSession: vi.fn(),
  streamChatMessage: vi.fn(),
  submitFeedback: vi.fn(),
}))

// Mock knowledge API
vi.mock('../../api/knowledge', () => ({
  listKnowledgeBasesForPortal: vi.fn(),
}))

describe('chat store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('initial state', () => {
    it('should have empty message list', () => {
      const store = useChatStore()
      expect(store.messages).toEqual([])
    })

    it('should have null current session', () => {
      const store = useChatStore()
      expect(store.currentSession).toBeNull()
    })

    it('should have no current step initially', () => {
      const store = useChatStore()
      expect(store.currentStep).toBe('')
    })

    it('should have null pipeline metrics initially', () => {
      const store = useChatStore()
      expect(store.pipelineMetrics).toBeNull()
    })

    it('should have default rag options', () => {
      const store = useChatStore()
      expect(store.ragOptions).toEqual({
        top_k: 5,
        query_rewrite: true,
        multi_route: true,
        hybrid: true,
        rerank: true,
      })
    })
  })

  describe('actions', () => {
    it('clearSession should reset all state', () => {
      const store = useChatStore()
      store.messages.push({ id: 'test-id', role: 'user', content: 'test' })
      store.currentStep = '混合检索'
      store.pipelineMetrics = { steps: [], total_duration_ms: 100 }

      store.clearSession()

      expect(store.messages).toEqual([])
      expect(store.currentSession).toBeNull()
      expect(store.currentStep).toBe('')
      expect(store.pipelineMetrics).toBeNull()
    })

    it('setCurrentStep should update step label', () => {
      const store = useChatStore()
      store.setCurrentStep('查询改写')
      expect(store.currentStep).toBe('查询改写')
    })
  })
})
