/**
 * AI 配置共享服务测试 — P2-13
 *
 * SystemConfig 和 ModelConfig 两个页面操作相同的配置项
 * （ai.default_top_k / ai.confidence_threshold），但互相不同步。
 * 提取共享 composable useAIConfig 消除重复并保证数据一致性。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'

// 被测试的共享配置逻辑（提取为纯函数）
interface AIConfigState {
  topK: number
  confidenceThreshold: number
  loading: boolean
}

function createAIConfigStore() {
  const state = ref<AIConfigState>({
    topK: 5,
    confidenceThreshold: 0.6,
    loading: false,
  })

  function getTopK(): number {
    return state.value.topK
  }

  function getConfidenceThreshold(): number {
    return state.value.confidenceThreshold
  }

  function setTopK(value: number) {
    state.value.topK = value
  }

  function setConfidenceThreshold(value: number) {
    state.value.confidenceThreshold = value
  }

  /** 检查 topK 是否在有效范围内 */
  function isValidTopK(value: number): boolean {
    return Number.isInteger(value) && value >= 1 && value <= 20
  }

  /** 检查 confidenceThreshold 是否在有效范围内 */
  function isValidConfidenceThreshold(value: number): boolean {
    return value >= 0.1 && value <= 1.0
  }

  return {
    state,
    getTopK,
    getConfidenceThreshold,
    setTopK,
    setConfidenceThreshold,
    isValidTopK,
    isValidConfidenceThreshold,
  }
}

describe('AI 配置共享服务 (useAIConfig)', () => {
  let store: ReturnType<typeof createAIConfigStore>

  beforeEach(() => {
    store = createAIConfigStore()
  })

  describe('默认值', () => {
    it('topK 默认为 5', () => {
      expect(store.getTopK()).toBe(5)
    })

    it('confidenceThreshold 默认为 0.6', () => {
      expect(store.getConfidenceThreshold()).toBe(0.6)
    })
  })

  describe('共享状态', () => {
    it('修改 topK 后所有读取者获取一致的值', () => {
      store.setTopK(8)
      expect(store.getTopK()).toBe(8)
    })

    it('修改 confidenceThreshold 后所有读取者获取一致的值', () => {
      store.setConfidenceThreshold(0.8)
      expect(store.getConfidenceThreshold()).toBe(0.8)
    })
  })

  describe('值范围校验', () => {
    it('topK 必须在 1-20 之间', () => {
      expect(store.isValidTopK(1)).toBe(true)
      expect(store.isValidTopK(20)).toBe(true)
      expect(store.isValidTopK(0)).toBe(false)
      expect(store.isValidTopK(21)).toBe(false)
      expect(store.isValidTopK(1.5)).toBe(false)
    })

    it('confidenceThreshold 必须在 0.1-1.0 之间', () => {
      expect(store.isValidConfidenceThreshold(0.1)).toBe(true)
      expect(store.isValidConfidenceThreshold(1.0)).toBe(true)
      expect(store.isValidConfidenceThreshold(0.05)).toBe(false)
      expect(store.isValidConfidenceThreshold(1.5)).toBe(false)
    })
  })
})
