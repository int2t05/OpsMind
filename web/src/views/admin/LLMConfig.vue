<template>
  <div class="llm-config-page">
    <div class="page-header">
      <h1 class="page-title">LLM 配置</h1>
      <button class="btn-add" @click="startCreate">新增配置</button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <p>加载中...</p>
    </div>

    <!-- 空状态 -->
    <div v-else-if="configs.length === 0" class="empty-state">
      <p>暂无 LLM 配置</p>
      <p class="empty-hint">点击"新增配置"添加 LLM 提供商（llama.cpp 或 OpenAI-compatible）</p>
    </div>

    <!-- 配置列表 -->
    <div v-else class="config-list">
      <div v-for="cfg in configs" :key="cfg.id" class="config-card">
        <div class="card-body">
          <div class="card-info">
            <div class="card-name">
              {{ cfg.name }}
              <span v-if="cfg.is_default" class="badge-default">默认</span>
            </div>
            <div class="card-meta">
              <span :class="['provider-tag', providerClass(cfg.provider_type)]">
                {{ providerText(cfg.provider_type) }}
              </span>
              <span class="meta-sep">·</span>
              <span>{{ cfg.llm_model }}</span>
              <span class="meta-sep">·</span>
              <span>Embedding: {{ cfg.embedding_model }}</span>
              <span class="meta-sep">·</span>
              <span>维度 {{ cfg.vector_dimension }}</span>
            </div>
            <div v-if="cfg.base_url" class="card-url">LLM: {{ cfg.base_url }}</div>
            <div v-if="cfg.embedding_base_url" class="card-url">Emb: {{ cfg.embedding_base_url }}</div>
          </div>
          <div class="card-actions">
            <button class="btn-action" @click="handleTestConnection(cfg)">测试连接</button>
            <button class="btn-action" @click="startEdit(cfg)">编辑</button>
            <button class="btn-action btn-action--danger" @click="confirmDelete(cfg)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <h2 class="modal-title">{{ editingId ? '编辑配置' : '新增配置' }}</h2>

        <div class="form-group">
          <label class="form-label">名称</label>
          <input v-model="form.name" class="form-input" placeholder="如 llama.cpp 本地、OpenAI" />
        </div>

        <div class="form-group">
          <label class="form-label">提供商类型</label>
          <select v-model.number="form.provider_type" class="form-select">
            <option :value="1">llama.cpp</option>
            <option :value="2">OpenAI-compatible</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">LLM Base URL</label>
          <input v-model="form.base_url" class="form-input" placeholder="http://llama-cpp:8080/v1" />
        </div>

        <div class="form-group">
          <label class="form-label">Embedding Base URL <span class="label-hint">（空则复用 LLM Base URL）</span></label>
          <input v-model="form.embedding_base_url" class="form-input" placeholder="留空则与 LLM Base URL 相同" />
        </div>

        <div class="form-group">
          <label class="form-label">API Key{{ form.provider_type === 1 ? '（llama.cpp 可留空）' : '' }}</label>
          <input v-model="form.api_key" class="form-input" placeholder="sk-..." />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">LLM 模型</label>
            <input v-model="form.llm_model" class="form-input" placeholder="qwen3-4b" />
          </div>
          <div class="form-group">
            <label class="form-label">Embedding 模型</label>
            <input v-model="form.embedding_model" class="form-input" placeholder="bge-m3" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Max Tokens</label>
            <input v-model.number="form.max_tokens" type="number" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label">向量维度</label>
            <input v-model.number="form.vector_dimension" type="number" class="form-input" />
          </div>
        </div>

        <div class="form-group">
          <label class="form-checkbox">
            <input type="checkbox" v-model="form.is_default" />
            <span>设为默认配置</span>
          </label>
        </div>

        <div class="modal-actions">
          <button class="btn-cancel" @click="closeModal">取消</button>
          <button class="btn-submit" :disabled="submitting" @click="handleSubmit">
            {{ submitting ? '提交中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="showDeleteConfirm" class="modal-overlay" @click.self="showDeleteConfirm = false">
      <div class="modal modal--confirm">
        <h2 class="modal-title">确认删除</h2>
        <p class="confirm-text">确定要删除 LLM 配置「{{ deleteTarget?.name }}」吗？此操作不可撤销。</p>
        <div class="modal-actions">
          <button class="btn-cancel" @click="showDeleteConfirm = false">取消</button>
          <button class="btn-submit btn-submit--danger" :disabled="deleting" @click="handleDelete">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 测试连接结果 -->
    <div v-if="showTestResult" class="modal-overlay" @click.self="showTestResult = false">
      <div class="modal modal--confirm">
        <h2 class="modal-title">连接测试</h2>
        <div v-if="testing" class="test-status">
          <p>正在测试连接...</p>
        </div>
        <div v-else-if="testResult" class="test-result">
          <div :class="['test-icon', testResult.success ? 'success' : 'failed']">
            {{ testResult.success ? '✓' : '✗' }}
          </div>
          <p class="test-message">
            {{ testResult.success ? `连接成功 — 模型: ${testResult.model}` : '连接失败' }}
          </p>
          <p v-if="testResult.success" class="test-latency">延迟: {{ testResult.latency_ms }}ms</p>
        </div>
        <div class="modal-actions">
          <button class="btn-cancel" @click="showTestResult = false">关闭</button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="toast.visible.value" :class="['toast', toast.type.value]">
      {{ toast.message.value }}
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/LLMConfig): 组件超过 610 行 — 应拆分为配置列表子组件、编辑弹窗子组件、连接测试组件。
// TODO(admin/LLMConfig): reactive() 和 ref() 混用管理表单状态 — form 用 reactive，deleteTarget/testResult 用 ref，
//                       增加了认知负担。建议统一使用 ref + 整体对象替换。
// TODO(admin/LLMConfig): Base URL 无 URL 格式校验；API Key 编辑模式下清空但无提示说明留空则保留原值。
import { ref, reactive, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import {
  getLLMConfigs, createLLMConfig, updateLLMConfig, deleteLLMConfig, testLLMConnection,
  type LLMConfigItem,
} from '@/api/llm_config'

const loading = ref(true)
const configs = ref<LLMConfigItem[]>([])
const showModal = ref(false)
const showDeleteConfirm = ref(false)
const showTestResult = ref(false)
const submitting = ref(false)
const deleting = ref(false)
const testing = ref(false)
const editingId = ref<number | null>(null)
const deleteTarget = ref<LLMConfigItem | null>(null)
const testResult = ref<{ success: boolean; model?: string; latency_ms?: number } | null>(null)
const toast = useToast()

const form = reactive({
  name: '',
  provider_type: 1,
  base_url: '',
  embedding_base_url: '',
  api_key: '',
  llm_model: '',
  embedding_model: '',
  max_tokens: 8192,
  vector_dimension: 1024,
  is_default: false,
})

onMounted(() => { fetchConfigs() })

// toast 已通过 useToast composable 管理，自动处理定时器清理

async function fetchConfigs() {
  loading.value = true
  try {
    const res = await getLLMConfigs()
    // TODO(admin/LLMConfig): getLLMConfigs 类型应直接返回 ApiResponse<LLMConfigItem[]> 解包后的 data。
    // 这里的 (res as any).data || res 是响应拦截器契约不清晰的表现。
    const data = (res as any).data || res
    configs.value = Array.isArray(data) ? data : (data?.items || [])
  } catch {
    configs.value = []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.name = ''
  form.provider_type = 1
  form.base_url = ''
  form.embedding_base_url = ''
  form.api_key = ''
  form.llm_model = ''
  form.embedding_model = ''
  form.max_tokens = 8192
  form.vector_dimension = 1024
  form.is_default = false
}

function startCreate() {
  editingId.value = null
  resetForm()
  showModal.value = true
}

function startEdit(cfg: LLMConfigItem) {
  editingId.value = cfg.id
  form.name = cfg.name
  form.provider_type = cfg.provider_type
  form.base_url = cfg.base_url
  form.embedding_base_url = cfg.embedding_base_url || ''
  form.api_key = ''  // API Key 不回显（后端返回脱敏值）
  form.llm_model = cfg.llm_model
  form.embedding_model = cfg.embedding_model
  form.max_tokens = cfg.max_tokens
  form.vector_dimension = cfg.vector_dimension
  form.is_default = cfg.is_default
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingId.value = null
}

function isValidURL(s: string): boolean {
  try { new URL(s); return true } catch { return false }
}

async function handleSubmit() {
  if (!form.name) { toast.showToast('名称不能为空', 'error'); return }
  if (!form.base_url) { toast.showToast('Base URL 不能为空', 'error'); return }
  if (!isValidURL(form.base_url)) { toast.showToast('Base URL 格式不正确', 'error'); return }
  if (form.embedding_base_url && !isValidURL(form.embedding_base_url)) { toast.showToast('Embedding Base URL 格式不正确', 'error'); return }
  if (form.provider_type === 1 && !form.base_url.includes('://')) { toast.showToast('容器内部署请使用完整 URL（如 http://llama-cpp:8080/v1）', 'error'); return }
  if (!form.llm_model) { toast.showToast('LLM 模型不能为空', 'error'); return }
  if (!form.embedding_model) { toast.showToast('Embedding 模型不能为空', 'error'); return }

  submitting.value = true
  try {
    const body = {
      name: form.name,
      provider_type: form.provider_type,
      base_url: form.base_url,
      embedding_base_url: form.embedding_base_url || '',
      api_key: form.api_key,
      llm_model: form.llm_model,
      embedding_model: form.embedding_model,
      max_tokens: form.max_tokens,
      vector_dimension: form.vector_dimension,
      is_default: form.is_default,
    }
    if (editingId.value) {
      await updateLLMConfig(editingId.value, body)
    } else {
      await createLLMConfig(body)
    }
    toast.showToast(editingId.value ? '更新成功' : '创建成功', 'success')
    closeModal()
    fetchConfigs()
  } catch (e: any) {
    toast.showToast(e?.response?.data?.message || e?.message || '操作失败', 'error')
  } finally {
    submitting.value = false
  }
}

async function handleTestConnection(cfg: LLMConfigItem) {
  // 如果正在编辑该配置且表单有改动，先保存再测试
  if (editingId.value && editingId.value === cfg.id) {
    await doSave()
  }
  showTestResult.value = true
  testing.value = true
  testResult.value = null
  try {
    const res = await testLLMConnection(cfg.id)
    const data = (res as any).data || res
    testResult.value = { success: true, model: data.model || cfg.llm_model, latency_ms: data.latency_ms || 0 }
  } catch (e: any) {
    testResult.value = { success: false }
  } finally {
    testing.value = false
  }
}

// 保存表单但不关闭对话框（供测试前自动保存使用）
async function doSave() {
  if (!form.name || !form.base_url || !form.llm_model || !form.embedding_model) return
  const body = {
    name: form.name, provider_type: form.provider_type, base_url: form.base_url,
    embedding_base_url: form.embedding_base_url || '', api_key: form.api_key,
    llm_model: form.llm_model, embedding_model: form.embedding_model,
    max_tokens: form.max_tokens, vector_dimension: form.vector_dimension, is_default: form.is_default,
  }
  try {
    await updateLLMConfig(editingId.value!, body)
  } catch { /* 静默保存，测试连接时会报告具体错误 */ }
}

function confirmDelete(cfg: LLMConfigItem) {
  deleteTarget.value = cfg
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deleteLLMConfig(deleteTarget.value.id)
    toast.showToast('删除成功', 'success')
    showDeleteConfirm.value = false
    fetchConfigs()
  } catch (e: any) {
    toast.showToast(e?.response?.data?.message || e?.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}

function providerClass(type: number) {
  return type === 1 ? 'llamacpp' : 'openai'
}

function providerText(type: number) {
  return type === 1 ? 'llama.cpp' : 'OpenAI'
}
</script>

<style scoped>
.llm-config-page {
  max-width: 900px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
}

.page-title {
  font-size: 22px;
  font-weight: 510;
  color: var(--text-primary);
}

.btn-add {
  padding: 8px 20px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  font-family: inherit;
}
.btn-add:hover { background: var(--accent-hover); }

.loading-state, .empty-state {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  font-size: 14px;
}

.empty-hint { font-size: 12px; margin-top: 6px; opacity: 0.6; }

/* 列表 */
.config-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.config-card {
  padding: 16px 20px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 10px;
}

.card-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.card-name {
  font-size: 15px;
  font-weight: 510;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge-default {
  font-size: 10px;
  padding: 2px 8px;
  background: rgba(94, 106, 210, 0.15);
  color: var(--accent);
  border-radius: 4px;
  font-weight: 500;
}

.card-meta {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
}

.card-url {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.7;
  margin-top: 4px;
  font-family: monospace;
}

.meta-sep { margin: 0 6px; }

.provider-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 3px;
  font-weight: 500;
}

.provider-tag.llamacpp {
  background: rgba(46, 160, 67, 0.15);
  color: var(--color-success);
}

.provider-tag.openai {
  background: rgba(16, 163, 127, 0.15);
  color: #10a37f;
}

.card-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
  margin-left: 16px;
}

.btn-action {
  padding: 5px 14px;
  background: var(--bg-base);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
}
.btn-action:hover { color: var(--text-primary); border-color: var(--border-hover); }
.btn-action--danger:hover { color: var(--tag-rejected-text); border-color: var(--tag-rejected-text); }

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  width: 520px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  padding: 28px;
  background: var(--bg-panel);
  border: 1px solid var(--border-default);
  border-radius: 12px;
}

.modal--confirm { width: 400px; }

.modal-title {
  font-size: 18px;
  font-weight: 510;
  color: var(--text-primary);
  margin-bottom: 22px;
}

.confirm-text {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 22px;
}

.form-group { margin-bottom: 16px; }

.form-label {
  display: block;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-input, .form-select {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-base);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
}
.label-hint { font-size: 11px; font-weight: 400; opacity: 0.6; }
.form-input:focus, .form-select:focus { outline: none; border-color: var(--accent); }

.form-row {
  display: flex;
  gap: 12px;
}

.form-row .form-group {
  flex: 1;
}

.form-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
}

.btn-cancel {
  padding: 8px 20px;
  background: var(--bg-base);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  font-family: inherit;
}

.btn-submit {
  padding: 8px 24px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  font-family: inherit;
}
.btn-submit:hover { background: var(--accent-hover); }
.btn-submit:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-submit--danger { background: var(--btn-danger-bg); }
.btn-submit--danger:hover { background: var(--btn-danger-bg); }

/* 测试结果 */
.test-status { text-align: center; padding: 20px; color: var(--text-secondary); font-size: 14px; }
.test-result { text-align: center; padding: 16px; }
.test-icon { font-size: 36px; margin-bottom: 12px; }
.test-icon.success { color: var(--btn-success-text); }
.test-icon.failed { color: var(--btn-danger-text); }
.test-message { font-size: 14px; color: var(--text-primary); margin-bottom: 6px; }
.test-latency { font-size: 12px; color: var(--text-secondary); }

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
