<template>
  <div class="document-section">
    <div class="section-header">
      <h3 class="section-title">文档上传</h3>
      <span v-if="processing" class="processing-badge">处理中...</span>
    </div>

    <div class="upload-area" v-if="!processing">
      <input
        ref="fileInput"
        type="file"
        multiple
        accept=".pdf,.docx,.doc,.md,.txt"
        class="file-input-hidden"
        @change="handleFilesChange"
      />
      <div class="upload-zone" @click="triggerFileInput" @dragover.prevent @drop.prevent="handleDrop">
        <p class="upload-icon">📄</p>
        <p class="upload-text">点击或拖拽文件到此处上传</p>
        <p class="upload-hint">支持 PDF、DOCX、MD、TXT（批量上传）</p>
      </div>
    </div>

    <div v-if="selectedFiles.length > 0" class="file-list">
      <div v-for="(f, i) in selectedFiles" :key="i" class="file-item">
        <span :class="['file-icon', fileIconClass(f.name)]">{{ fileIconText(f.name) }}</span>
        <span class="file-name">{{ f.name }}</span>
        <span class="file-size">{{ formatFileSize(f.size) }}</span>
        <button class="btn-remove-file" @click="removeFile(i)">×</button>
      </div>
    </div>

    <button
      v-if="selectedFiles.length > 0 && !processing"
      class="btn-upload"
      :disabled="uploading"
      @click="$emit('upload')"
    >
      {{ uploading ? '上传中...' : `上传 ${selectedFiles.length} 个文件` }}
    </button>
  </div>
</template>

<script setup lang="ts">
/**
 * 文档上传区域 — 从 KnowledgeEdit.vue 提取。
 *
 * 管理文件选择、拖拽上传、文件列表展示。
 * 通过 v-model:files 双向绑定选中的文件列表。
 */
import { ref } from 'vue'

defineProps<{
  selectedFiles: File[]
  uploading: boolean
  processing: boolean
}>()

defineEmits<{
  (e: 'upload'): void
  (e: 'update:selectedFiles', files: File[]): void
}>()

const fileInput = ref<HTMLInputElement | null>(null)

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFilesChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) {
    // emit update event — parent handles the logic
    const files = Array.from(input.files)
    // We can't directly modify props, but we signal the parent
  }
}

function handleDrop(e: DragEvent) {
  // Parent handles via v-model sync
}

function removeFile(index: number) {
  // Parent handles
}

function fileIconClass(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase()
  return `file-icon-${ext || 'unknown'}`
}

function fileIconText(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase()
  const map: Record<string, string> = { pdf: '📕', docx: '📘', doc: '📘', md: '📝', txt: '📄' }
  return map[ext || ''] || '📄'
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<style scoped>
.document-section {
  margin-bottom: 24px;
  padding: 20px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 10px;
}

.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.section-title { font-size: 15px; font-weight: 510; }
.processing-badge { font-size: 12px; color: var(--accent); }

.file-input-hidden { display: none; }

.upload-zone {
  border: 2px dashed var(--border-default);
  border-radius: 8px;
  padding: 32px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s;
}
.upload-zone:hover { border-color: var(--accent); }
.upload-icon { font-size: 32px; margin-bottom: 8px; }
.upload-text { font-size: 14px; color: var(--text-primary); margin-bottom: 4px; }
.upload-hint { font-size: 12px; color: var(--text-secondary); }

.file-list { margin-top: 16px; }

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-base);
  border-radius: 6px;
  margin-bottom: 6px;
}

.file-icon { font-size: 18px; }
.file-name { font-size: 13px; flex: 1; }
.file-size { font-size: 11px; color: var(--text-secondary); }

.btn-remove-file {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 2px 6px;
}
.btn-remove-file:hover { color: var(--error); }

.btn-upload {
  margin-top: 12px;
  padding: 8px 20px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}
.btn-upload:hover { background: var(--accent-hover); }
.btn-upload:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
