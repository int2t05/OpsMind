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
// TODO(components/Pagination): 组件仅 emit update:current-page 和 update:page-size，但多个调用方
//                           使用 @change 监听（该事件不存在）— portal/Messages.vue 和 portal/TicketQuery.vue
//                           分页功能因此失效。需统一改为 v-model:current-page 或修正事件名。
import { NPagination } from 'naive-ui'

defineProps<{ total: number; currentPage: number; pageSize: number }>()
defineEmits<{
  (e: 'update:current-page', page: number): void
  (e: 'update:page-size', size: number): void
}>()
</script>
