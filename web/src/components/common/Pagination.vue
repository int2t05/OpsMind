<template>
  <div class="pagination">
    <button :disabled="currentPage <= 1" @click="changePage(currentPage - 1)">上一页</button>
    <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
    <button :disabled="currentPage >= totalPages" @click="changePage(currentPage + 1)">下一页</button>
    <select v-model="pageSize" @change="onPageSizeChange">
      <option :value="10">10条/页</option>
      <option :value="20">20条/页</option>
      <option :value="50">50条/页</option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  total: number
  currentPage: number
  pageSize: number
}>()

const emit = defineEmits<{
  (e: 'update:currentPage', page: number): void
  (e: 'update:pageSize', size: number): void
}>()

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))

const changePage = (page: number) => {
  emit('update:currentPage', page)
}

const onPageSizeChange = (event: Event) => {
  const target = event.target as HTMLSelectElement
  emit('update:pageSize', Number(target.value))
}
</script>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
}

button {
  padding: 6px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-primary);
  border-radius: 4px;
  cursor: pointer;
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

select {
  padding: 6px 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-primary);
  border-radius: 4px;
}
</style>
