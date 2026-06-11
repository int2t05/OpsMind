<template>
  <n-pagination
    :page="currentPage"
    :page-size="pageSize"
    :item-count="total"
    :page-sizes="[10, 20, 50]"
    show-size-picker
    @update:page="$emit('update:current-page', $event)"
    @update:page-size="$emit('update:page-size', $event)"
  />
</template>

<script setup lang="ts">
import { NPagination } from 'naive-ui'

defineProps<{ total: number; currentPage: number; pageSize: number }>()
defineEmits<{
  (e: 'update:current-page', page: number): void
  (e: 'update:page-size', size: number): void
}>()
</script>
