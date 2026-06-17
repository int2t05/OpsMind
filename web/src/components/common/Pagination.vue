<template>
  <n-pagination
    :page="currentPage"
    :page-size="pageSize"
    :item-count="total"
    :page-sizes="[10, 20, 50]"
    show-size-picker
    @update:page="$emit('update:current-page', $event)"
    @update:page-size="onPageSizeChange"
  />
</template>

<script setup lang="ts">
import { NPagination } from 'naive-ui'

const props = defineProps<{ total: number; currentPage: number; pageSize: number }>()
const emit = defineEmits<{
  (e: 'update:current-page', page: number): void
  (e: 'update:page-size', size: number): void
}>()

// page-size 改变时重置 currentPage，避免从高页码切大 pageSize 后请求到空页
function onPageSizeChange(size: number) {
  emit('update:page-size', size)
  if (props.currentPage !== 1) {
    emit('update:current-page', 1)
  }
}
</script>
