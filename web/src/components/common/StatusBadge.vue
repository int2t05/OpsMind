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

// 状态文本映射
const TEXT_MAP: Record<string, Record<number, string>> = {
  user: { 1: '正常', 2: '已冻结' },
  ticket: { 0: '待处理', 1: '处理中', 2: '需补充信息', 3: '已解决', 4: '已关闭' },
  knowledge: { 0: '草稿', 1: '待审核', 2: '已发布', 3: '已停用' },
}
// Naive UI Tag type 映射
const TYPE_MAP: Record<string, Record<number, 'success' | 'error' | 'warning' | 'info' | 'default'>> = {
  user: { 1: 'success', 2: 'error' },
  ticket: { 0: 'warning', 1: 'info', 2: 'warning', 3: 'success', 4: 'default' },
  knowledge: { 0: 'default', 1: 'warning', 2: 'success', 3: 'error' },
}

const statusText = computed(() => TEXT_MAP[props.type]?.[props.status] || '未知')
const tagType = computed(() => TYPE_MAP[props.type]?.[props.status] || 'default')
</script>
