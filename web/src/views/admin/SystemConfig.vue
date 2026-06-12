<template>
  <div class="system-config-page">
    <div class="page-header">
      <h1 class="page-title">系统配置</h1>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <p>加载配置中...</p>
    </div>

    <!-- 配置为空 -->
    <div v-else-if="configItems.length === 0" class="empty-state">
      <p>暂无可配置项</p>
    </div>

    <!-- 配置列表 -->
    <div v-else class="config-table-wrapper">
      <table class="config-table">
        <thead>
          <tr>
            <th>配置项</th>
            <th>当前值</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in configItems" :key="item.key">
            <td>
              <div class="key-name">{{ item.key }}</div>
              <div class="key-desc" v-if="item.key === 'ai.default_top_k'">RAG 检索 Top K 数量</div>
              <div class="key-desc" v-else-if="item.key === 'ai.confidence_threshold'">AI 置信度阈值</div>
            </td>
            <td class="value-cell">
              <span v-if="editingKey !== item.key" class="value-display">{{ formatValue(item.value) }}</span>
              <input
                v-else
                v-model="editValue"
                class="value-input"
                :type="typeof (item as any)._parsed === 'number' ? 'number' : 'text'"
              />
            </td>
            <td class="action-cell">
              <template v-if="editingKey !== item.key">
                <button class="btn-edit" @click="startEdit(item)">编辑</button>
              </template>
              <template v-else>
                <button
                  class="btn-save-inline"
                  :disabled="saving"
                  @click="handleSave(item.key)"
                >
                  {{ saving ? '保存中' : '保存' }}
                </button>
                <button class="btn-cancel-inline" @click="cancelEdit">取消</button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Toast 提示 -->
    <div v-if="toast.visible.value" :class="['toast', toast.type.value]">
      {{ toast.message.value }}
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/SystemConfig): 与 ModelConfig 页面管理完全相同的配置项（ai.default_top_k / ai.confidence_threshold），
//                         修改一个不会同步到另一个 — 应合并为统一的 AI 配置入口或共享配置读写逻辑。
// TODO(admin/SystemConfig): toast 定时器未在 onUnmounted 清理 — 存在内存泄漏。
import { ref, onMounted, watch } from 'vue'
import { useAIConfig } from '@/composables/useAIConfig'
import { useToast } from '@/composables/useToast'
import { getConfig, setConfig } from '@/api/config'

interface ConfigItem {
  key: string
  value: any
  _parsed?: any
}

const aiConfig = useAIConfig()
const toast = useToast()
const saving = ref(false)
const configItems = ref<ConfigItem[]>([])
const editingKey = ref('')
const editValue = ref<any>('')
const loading = aiConfig.loading

// 可配置的系统配置项
const KNOWN_KEYS = ['ai.default_top_k', 'ai.confidence_threshold']

onMounted(async () => {
  await aiConfig.loadConfig(getConfig)
  syncToItems()
})

// 当 AI 配置变化时同步到配置列表
watch([aiConfig.topK, aiConfig.confidenceThreshold], () => {
  syncToItems()
})

/** 将共享状态同步到本地配置项列表 */
function syncToItems() {
  configItems.value = [
    { key: 'ai.default_top_k', value: aiConfig.topK.value, _parsed: aiConfig.topK.value },
    { key: 'ai.confidence_threshold', value: aiConfig.confidenceThreshold.value, _parsed: aiConfig.confidenceThreshold.value },
  ]
}

function formatValue(val: any): string {
  if (val === null || val === undefined) return '(未设置)'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

function startEdit(item: ConfigItem) {
  editingKey.value = item.key
  editValue.value = item._parsed ?? item.value
}

function cancelEdit() {
  editingKey.value = ''
  editValue.value = ''
}

async function handleSave(key: string) {
  saving.value = true
  try {
    // 尝试解析数字值
    let value: any = editValue.value
    const num = Number(value)
    if (!isNaN(num) && String(value).trim() !== '') value = num

    await setConfig(key, value)
    toast.showToast('保存成功', 'success')
    editingKey.value = ''

    // 同步到共享 AI 配置状态
    if (key === 'ai.default_top_k') aiConfig.setTopK(Number(value))
    if (key === 'ai.confidence_threshold') aiConfig.setConfidenceThreshold(Number(value))
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '保存失败'
    toast.showToast(msg, 'error')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.system-config-page {
  max-width: 720px;
}

.page-header { margin-bottom: 28px; }
.page-title {
  font-size: 22px;
  font-weight: 510;
  color: var(--text-primary);
}

.loading-state, .empty-state {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  font-size: 14px;
}

/* 表格 */
.config-table-wrapper {
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  overflow: hidden;
}

.config-table {
  width: 100%;
  border-collapse: collapse;
}

.config-table th {
  text-align: left;
  padding: 12px 20px;
  font-size: 12px;
  font-weight: 510;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border-default);
  background: var(--bg-base);
}

.config-table td {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-default);
  font-size: 14px;
  color: var(--text-primary);
}

.config-table tr:last-child td { border-bottom: none; }

.key-name {
  font-weight: 510;
}

.key-desc {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 3px;
}

.value-display {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.value-input {
  padding: 6px 10px;
  background: var(--bg-base);
  border: 1px solid var(--accent);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  width: 120px;
  font-family: inherit;
}
.value-input:focus { outline: none; }

.action-cell {
  text-align: right;
}

.btn-edit {
  padding: 5px 14px;
  background: var(--bg-base);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
}
.btn-edit:hover { color: var(--text-primary); border-color: var(--border-hover); }

.btn-save-inline {
  padding: 5px 14px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
  margin-right: 6px;
}
.btn-save-inline:hover { background: var(--accent-hover); }
.btn-save-inline:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-cancel-inline {
  padding: 5px 14px;
  background: var(--bg-base);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
}

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

.toast.success { background: var(--toast-success-bg); color: var(--toast-success-text); border: 1px solid var(--toast-success-border); }
.toast.error { background: var(--toast-error-bg); color: var(--toast-error-text); border: 1px solid var(--toast-error-border); }

@keyframes slideIn {
  from { transform: translateY(20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
</style>
