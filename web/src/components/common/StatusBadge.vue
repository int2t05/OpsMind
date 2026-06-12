<template>
  <n-tag :type="tagType" :bordered="false" size="small" round>
    {{ statusText }}
  </n-tag>
</template>

<script setup lang="ts">
// TODO(components/StatusBadge): knowledge 状态映射 `{ 0:'草稿', 1:'待审核', 2:'已发布', 3:'已停用' }`
//                             与实际后端状态码 0-5（停用/草稿/待审核/已通过/已发布/已驳回）不一致，
//                             此类型分支也未被任何知识库视图使用。需同步更新映射或删除该分支。
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = withDefaults(defineProps<{ status: number; type?: 'user' | 'ticket' | 'knowledge' }>(), {
  type: 'user',
})

// 状态文本映射
const TEXT_MAP: Record<string, Record<number, string>> = {
  user: { 1: '正常', 2: '已冻结' },
  ticket: { 0: '待处理', 1: '处理中', 2: '需补充信息', 3: '已解决', 4: '已关闭' },
}
// Naive UI Tag type 映射
const TYPE_MAP: Record<string, Record<number, 'success' | 'error' | 'warning' | 'info' | 'default'>> = {
  user: { 1: 'success', 2: 'error' },
  ticket: { 0: 'warning', 1: 'info', 2: 'warning', 3: 'success', 4: 'default' },
}

const statusText = computed(() => TEXT_MAP[props.type]?.[props.status] || '未知')
const tagType = computed(() => TYPE_MAP[props.type]?.[props.status] || 'default')
</script>
