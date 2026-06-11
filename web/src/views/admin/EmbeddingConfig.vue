<template>
  <div class="embedding-config-page">
    <div class="page-header">
      <h1 class="page-title">Embedding 配置</h1>
      <button class="btn-add" @click="startCreate">新增配置</button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <p>加载中...</p>
    </div>

    <!-- 空状态 -->
    <div v-else-if="configs.length === 0" class="empty-state">
      <p>暂无 Embedding 配置</p>
      <p class="empty-hint">点击"新增配置"创建第一个 Embedding 模型配置</p>
    </div>

    <!-- 配置列表 -->
    <div v-else class="config-list">
      <div v-for="cfg in configs" :key="cfg.id" class="config-card">
        <div class="card-body">
          <div class="card-info">
            <div class="card-name">
              {{ cfg.name || cfg.model_name }}
              <span v-if="cfg.is_default" class="badge-default">默认</span>
            </div>
            <div class="card-meta">
              <span>{{ cfg.model_type === 1 ? 'API 类型' : '本地模型' }}</span>
              <span class="meta-sep">·</span>
              <span>维度 {{ cfg.vector_dimension }}</span>
            </div>
          </div>
          <div class="card-actions">
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
          <label class="form-label">模型名称</label>
          <input
            v-model="form.modelName"
            class="form-input"
            placeholder="如 text-embedding-ada-002"
          />
        </div>

        <div class="form-group">
          <label class="form-label">模型类型</label>
          <select v-model.number="form.modelType" class="form-select">
            <option :value="1">API 类型</option>
            <option :value="2">本地模型</option>
          </select>
        </div>

        <div class="form-group" v-if="form.modelType === 1">
          <label class="form-label">API 地址</label>
          <input v-model="form.apiEndpoint" class="form-input" placeholder="https://api.example.com/v1/embeddings" />
        </div>

        <div class="form-group" v-if="form.modelType === 2">
          <label class="form-label">本地模型路径</label>
          <input v-model="form.localPath" class="form-input" placeholder="/models/bge-large-zh" />
        </div>

        <div class="form-group">
          <label class="form-label">向量维度</label>
          <input v-model.number="form.vectorDimension" type="number" class="form-input" placeholder="1536" />
        </div>

        <div class="form-group">
          <label class="form-checkbox">
            <input type="checkbox" v-model="form.isDefault" />
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
        <p class="confirm-text">确定要删除 Embedding 配置「{{ deleteTarget?.name || deleteTarget?.model_name }}」吗？此操作不可撤销。</p>
        <div class="modal-actions">
          <button class="btn-cancel" @click="showDeleteConfirm = false">取消</button>
          <button class="btn-submit btn-submit--danger" :disabled="deleting" @click="handleDelete">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="toast.message" :class="['toast', toast.type]">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

interface EmbeddingConfigItem {
  id: number
  name?: string
  model_name?: string
  model_type: number
  api_endpoint?: string
  local_path?: string
  vector_dimension: number
  is_default: boolean
}

const loading = ref(true)
const configs = ref<EmbeddingConfigItem[]>([])
const showModal = ref(false)
const showDeleteConfirm = ref(false)
const submitting = ref(false)
const deleting = ref(false)
const editingId = ref<number | null>(null)
const deleteTarget = ref<EmbeddingConfigItem | null>(null)
const toast = ref<{ message: string; type: string }>({ message: '', type: 'success' })
let toastTimer: ReturnType<typeof setTimeout> | null = null

const form = ref({
  modelName: '',
  modelType: 1,
  apiEndpoint: '',
  localPath: '',
  vectorDimension: 1536,
  isDefault: false
})

onMounted(() => { fetchConfigs() })

function showToast(message: string, type: 'success' | 'error') {
  toast.value = { message, type }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = { message: '', type: 'success' } }, 3000)
}

async function fetchConfigs() {
  loading.value = true
  try {
    const res = await request.get('/api/v1/admin/embedding-configs' as any)
    const data = (res as any).data || res
    configs.value = Array.isArray(data) ? data : (data?.items || [])
  } catch {
    configs.value = []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { modelName: '', modelType: 1, apiEndpoint: '', localPath: '', vectorDimension: 1536, isDefault: false }
}

function startCreate() {
  editingId.value = null
  resetForm()
  showModal.value = true
}

function startEdit(cfg: EmbeddingConfigItem) {
  editingId.value = cfg.id
  form.value = {
    modelName: cfg.name || cfg.model_name || '',
    modelType: cfg.model_type,
    apiEndpoint: cfg.api_endpoint || '',
    localPath: cfg.local_path || '',
    vectorDimension: cfg.vector_dimension,
    isDefault: cfg.is_default
  }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingId.value = null
}

async function handleSubmit() {
  submitting.value = true
  try {
    const body = {
      model_name: form.value.modelName,
      model_type: form.value.modelType,
      api_endpoint: form.value.apiEndpoint,
      local_path: form.value.localPath,
      vector_dimension: form.value.vectorDimension,
      is_default: form.value.isDefault
    }
    if (editingId.value) {
      await request.put(`/api/v1/admin/embedding-configs/${editingId.value}` as any, body)
    } else {
      await request.post('/api/v1/admin/embedding-configs' as any, body)
    }
    showToast(editingId.value ? '更新成功' : '创建成功', 'success')
    closeModal()
    fetchConfigs()
  } catch (e: any) {
    showToast(e?.response?.data?.message || e?.message || '操作失败', 'error')
  } finally {
    submitting.value = false
  }
}

function confirmDelete(cfg: EmbeddingConfigItem) {
  deleteTarget.value = cfg
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await request.delete(`/api/v1/admin/embedding-configs/${deleteTarget.value.id}` as any)
    showToast('删除成功', 'success')
    showDeleteConfirm.value = false
    fetchConfigs()
  } catch (e: any) {
    showToast(e?.response?.data?.message || e?.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.embedding-config-page {
  max-width: 800px;
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
  align-items: center;
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
}

.meta-sep { margin: 0 6px; }

.card-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
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
  width: 460px;
  max-width: 90vw;
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
.form-input:focus, .form-select:focus { outline: none; border-color: var(--accent); }

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
