<template>
  <div class="model-config-page">
    <div class="page-header">
      <h1 class="page-title">模型配置</h1>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <p>加载配置中...</p>
    </div>

    <!-- v2: LLM 配置入口 -->
    <div class="llm-config-link">
      <router-link to="/admin/llm-config" class="link-card">
        <div class="link-title">LLM 配置</div>
        <div class="link-desc">管理 LLM 和 Embedding 提供商（llama.cpp / OpenAI-compatible）</div>
        <span class="link-arrow">→</span>
      </router-link>
    </div>

    <!-- 配置表单 -->
    <div v-if="!loading" class="config-form">
      <!-- Top K -->
      <div class="config-item">
        <div class="config-header">
          <label class="config-label">默认 Top K</label>
          <span class="config-value">{{ topK }}</span>
        </div>
        <p class="config-desc">RAG 检索时返回的最相关文档数量（1-10）</p>
        <input
          type="range"
          v-model.number="topK"
          min="1"
          max="10"
          class="slider"
        />
        <div class="slider-marks">
          <span>1</span><span>5</span><span>10</span>
        </div>
      </div>

      <!-- 置信度阈值 -->
      <div class="config-item">
        <div class="config-header">
          <label class="config-label">置信度阈值</label>
          <span class="config-value">{{ confidenceThreshold }}</span>
        </div>
        <p class="config-desc">
          低于此值的 AI 回答将建议用户提交申告（0.1-1.0）
        </p>
        <input
          type="range"
          v-model.number="confidenceThreshold"
          min="0.1"
          max="1.0"
          step="0.05"
          class="slider"
        />
        <div class="slider-marks">
          <span>0.1</span><span>0.5</span><span>1.0</span>
        </div>
      </div>

      <!-- 保存按钮 -->
      <div class="form-actions">
        <button
          class="btn-save"
          :disabled="saving"
          @click="handleSave"
        >
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>

      <!-- Toast 提示 -->
      <div v-if="toast.message" :class="['toast', toast.type]">
        {{ toast.message }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/ModelConfig): 与 SystemConfig 页面管理完全相同的配置项 — 应合并为统一的 AI 配置入口。
// TODO(admin/ModelConfig): 组件超过 310 行 — 可提取滑块配置子组件。
// TODO(admin/ModelConfig): toast 定时器未在 onUnmounted 清理 — 存在内存泄漏。
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

const loading = ref(true)
const saving = ref(false)
const topK = ref(5)
const confidenceThreshold = ref(0.6)
const toast = ref<{ message: string; type: string }>({ message: '', type: 'success' })

let toastTimer: ReturnType<typeof setTimeout> | null = null

onMounted(async () => {
  try {
    const [topKRes, thresholdRes] = await Promise.all([
      request.get('/api/v1/admin/configs/ai.default_top_k' as any),
      request.get('/api/v1/admin/configs/ai.confidence_threshold' as any)
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
})

function showToast(message: string, type: 'success' | 'error') {
  toast.value = { message, type }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = { message: '', type: 'success' } }, 3000)
}

async function handleSave() {
  saving.value = true
  try {
    await Promise.all([
      request.put('/api/v1/admin/configs/ai.default_top_k' as any, { value: topK.value }),
      request.put('/api/v1/admin/configs/ai.confidence_threshold' as any, { value: confidenceThreshold.value })
    ])
    showToast('配置保存成功', 'success')
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '保存失败'
    showToast(msg, 'error')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.model-config-page {
  max-width: 640px;
}

.page-header { margin-bottom: 28px; }
.page-title {
  font-size: 22px;
  font-weight: 510;
  color: var(--text-primary);
}

/* v2: LLM 配置入口链接 */
.llm-config-link { margin-bottom: 24px; }

.link-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  text-decoration: none;
  transition: border-color 0.2s;
}

.link-card:hover { border-color: var(--accent); }

.link-title {
  font-size: 15px;
  font-weight: 510;
  color: var(--accent);
  flex-shrink: 0;
}

.link-desc {
  font-size: 13px;
  color: var(--text-secondary);
  flex: 1;
}

.link-arrow {
  font-size: 16px;
  color: var(--text-secondary);
}

.loading-state {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  font-size: 14px;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.config-item {
  padding: 20px 24px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 10px;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.config-label {
  font-size: 15px;
  font-weight: 510;
  color: var(--text-primary);
}

.config-value {
  font-size: 18px;
  font-weight: 510;
  color: var(--accent);
  min-width: 36px;
  text-align: right;
}

.config-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 14px;
}

.slider {
  width: 100%;
  height: 6px;
  -webkit-appearance: none;
  appearance: none;
  background: var(--bg-base);
  border-radius: 3px;
  outline: none;
}

.slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--accent);
  cursor: pointer;
}

.slider::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--accent);
  cursor: pointer;
  border: none;
}

.slider-marks {
  display: flex;
  justify-content: space-between;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-secondary);
}

.form-actions {
  padding-top: 4px;
}

.btn-save {
  padding: 10px 32px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
}

.btn-save:hover { background: var(--accent-hover); }
.btn-save:disabled { opacity: 0.5; cursor: not-allowed; }

/* Toast */
.toast {
  position: fixed;
  bottom: 32px;
  right: 32px;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 14px;
  z-index: 9999;
  animation: slideIn 0.3s ease;
}

.toast.success {
  background: var(--toast-success-bg);
  color: var(--toast-success-text);
  border: 1px solid var(--toast-success-border);
}

.toast.error {
  background: var(--toast-error-bg);
  color: var(--toast-error-text);
  border: 1px solid var(--toast-error-border);
}

@keyframes slideIn {
  from { transform: translateY(20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
</style>
