<template>
  <span :class="['status-badge', statusClass]">{{ statusText }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: number
  type?: 'user' | 'ticket' | 'knowledge'
}>()

const statusConfig: Record<string, Record<number, { text: string; class: string }>> = {
  user: {
    1: { text: '正常', class: 'active' },
    2: { text: '已冻结', class: 'frozen' },
  },
  ticket: {
    0: { text: '待处理', class: 'pending' },
    1: { text: '处理中', class: 'processing' },
    2: { text: '需补充信息', class: 'info' },
    3: { text: '已解决', class: 'resolved' },
    4: { text: '已关闭', class: 'closed' },
  },
  knowledge: {
    0: { text: '草稿', class: 'draft' },
    1: { text: '待审核', class: 'pending' },
    2: { text: '已发布', class: 'published' },
    3: { text: '已停用', class: 'disabled' },
  },
}

const config = computed(() => {
  const type = props.type || 'user'
  return statusConfig[type]?.[props.status] || { text: '未知', class: 'unknown' }
})

const statusText = computed(() => config.value.text)
const statusClass = computed(() => config.value.class)
</script>

<style scoped>
.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.active, .published, .resolved { background: #1a3a2a; color: #4ade80; }
.frozen, .disabled, .closed { background: #3a1a1a; color: #f87171; }
.pending, .draft { background: #3a3a1a; color: #fbbf24; }
.processing, .info { background: #1a2a3a; color: #60a5fa; }
.unknown { background: #2a2a2a; color: #9ca3af; }
</style>
