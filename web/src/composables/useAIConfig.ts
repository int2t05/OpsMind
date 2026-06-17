/**
 * AI 配置共享状态管理
 *
 * SystemConfig 和 ModelConfig 两个页面操作相同的配置项
 * （ai.default_top_k / ai.confidence_threshold），之前各自独立读取和保存，
 * 修改一个页面不会同步到另一个。
 *
 * 此 composable 提供共享的响应式状态，两个页面引用同一个数据源。
 */
import { ref } from 'vue'

/** AI 配置的默认值 */
const DEFAULT_TOP_K = 5
const DEFAULT_CONFIDENCE_THRESHOLD = 0.6

// 模块级共享状态 — 所有使用此 composable 的组件共享同一份数据
const topK = ref(DEFAULT_TOP_K)
const confidenceThreshold = ref(DEFAULT_CONFIDENCE_THRESHOLD)
const loading = ref(false)

export function useAIConfig() {
  /** 检查 topK 是否在有效范围内 */
  function isValidTopK(value: number): boolean {
    return Number.isInteger(value) && value >= 1 && value <= 20
  }

  /** 检查 confidenceThreshold 是否在有效范围内 */
  function isValidConfidenceThreshold(value: number): boolean {
    return value >= 0.1 && value <= 1.0
  }

  /** 设置 topK（仅在有效范围内） */
  function setTopK(value: number) {
    if (isValidTopK(value)) {
      topK.value = value
    }
  }

  /** 设置 confidenceThreshold（仅在有效范围内） */
  function setConfidenceThreshold(value: number) {
    if (isValidConfidenceThreshold(value)) {
      confidenceThreshold.value = value
    }
  }

  /** 从后端加载配置值 */
  async function loadConfig(getConfig: (key: string) => Promise<{ data: unknown }>) {
    // TODO(composable/ai-config): loadConfig 吞掉错误并使用默认值，页面无法提示配置加载失败。
    // 应返回错误状态或让调用方决定是否展示降级提示。
    loading.value = true
    try {
      const [topKRes, thresholdRes] = await Promise.all([
        getConfig('ai.default_top_k'),
        getConfig('ai.confidence_threshold'),
      ])
      const tk = (topKRes as any).data ?? topKRes
      const ct = (thresholdRes as any).data ?? thresholdRes
      if (tk != null) topK.value = Number(tk)
      if (ct != null) confidenceThreshold.value = Number(ct)
    } catch {
      // 使用默认值
    } finally {
      loading.value = false
    }
  }

  return {
    topK,
    confidenceThreshold,
    loading,
    setTopK,
    setConfidenceThreshold,
    isValidTopK,
    isValidConfidenceThreshold,
    loadConfig,
  }
}
