<template>
  <n-tag :type="tagType" :bordered="false" size="small" round>
    {{ statusText }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = withDefaults(defineProps<{ status: number; type?: 'user' | 'ticket' | 'knowledge' }>(), {
  type: 'user',
})

// 状态文本映射（1-based，与后端 model 对齐）
const TEXT_MAP: Record<string, Record<number, string>> = {
  user: { 1: '正常', 2: '已冻结' },
  ticket: { 1: '待处理', 2: '处理中', 3: '需补充信息', 4: '已解决', 5: '已关闭' },
}
// Naive UI Tag type 映射
const TYPE_MAP: Record<string, Record<number, 'success' | 'error' | 'warning' | 'info' | 'default'>> = {
  user: { 1: 'success', 2: 'error' },
  ticket: { 1: 'warning', 2: 'info', 3: 'warning', 4: 'success', 5: 'default' },
}

const statusText = computed(() => TEXT_MAP[props.type]?.[props.status] || '未知')
const tagType = computed(() => TYPE_MAP[props.type]?.[props.status] || 'default')
</script>
