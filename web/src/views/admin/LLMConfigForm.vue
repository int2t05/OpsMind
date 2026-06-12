<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <h2 class="modal-title">{{ isEdit ? '编辑配置' : '新增配置' }}</h2>

      <div class="form-group">
        <label class="form-label">名称</label>
        <input v-model="local.name" class="form-input" placeholder="如 llama.cpp 本地、OpenAI" />
      </div>

      <div class="form-group">
        <label class="form-label">提供商类型</label>
        <select v-model.number="local.provider_type" class="form-select">
          <option :value="1">llama.cpp</option>
          <option :value="2">OpenAI-compatible</option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">LLM Base URL</label>
        <input v-model="local.base_url" class="form-input" placeholder="http://llama-cpp:8080/v1" />
      </div>

      <div class="form-group">
        <label class="form-label">Embedding Base URL <span class="label-hint">（空则复用 LLM Base URL）</span></label>
        <input v-model="local.embedding_base_url" class="form-input" placeholder="留空则与 LLM Base URL 相同" />
      </div>

      <div class="form-group">
        <label class="form-label">API Key{{ local.provider_type === 1 ? '（llama.cpp 可留空）' : '' }}</label>
        <input v-model="local.api_key" class="form-input" placeholder="sk-..." />
      </div>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label">LLM 模型</label>
          <input v-model="local.llm_model" class="form-input" placeholder="qwen3-4b" />
        </div>
        <div class="form-group">
          <label class="form-label">Embedding 模型</label>
          <input v-model="local.embedding_model" class="form-input" placeholder="bge-m3" />
        </div>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label">Max Tokens</label>
          <input v-model.number="local.max_tokens" type="number" class="form-input" />
        </div>
        <div class="form-group">
          <label class="form-label">向量维度</label>
          <input v-model.number="local.vector_dimension" type="number" class="form-input" />
        </div>
      </div>

      <div class="form-group">
        <label class="form-checkbox">
          <input type="checkbox" v-model="local.is_default" />
          <span>设为默认配置</span>
        </label>
      </div>

      <div class="modal-actions">
        <button class="btn-cancel" @click="$emit('close')">取消</button>
        <button class="btn-submit" :disabled="submitting" @click="$emit('submit', { ...local })">
          {{ submitting ? '提交中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * LLM 配置表单弹窗 — 从 LLMConfig.vue 提取，降低父组件复杂度。
 *
 * 通过 v-model 双向绑定表单状态，通过 emit 提交事件通知父组件。
 */
import { reactive, watch, ref } from 'vue'
import type { LLMConfigItem } from '@/api/llm_config'

const props = defineProps<{
  visible: boolean
  editingConfig: LLMConfigItem | null
  submitting: boolean
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'submit', form: typeof local): void
}>()

const isEdit = ref(false)

const local = reactive({
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

// 当 editingConfig 变化时同步表单
watch(
  () => props.editingConfig,
  (cfg) => {
    if (cfg) {
      isEdit.value = true
      Object.assign(local, {
        name: cfg.name || '',
        provider_type: cfg.provider_type ?? 1,
        base_url: cfg.base_url || '',
        embedding_base_url: cfg.embedding_base_url || '',
        api_key: '',
        llm_model: cfg.llm_model || '',
        embedding_model: cfg.embedding_model || '',
        max_tokens: cfg.max_tokens ?? 8192,
        vector_dimension: cfg.vector_dimension ?? 1024,
        is_default: cfg.is_default ?? false,
      })
    } else {
      isEdit.value = false
      Object.assign(local, {
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
    }
  },
  { immediate: true },
)
</script>
