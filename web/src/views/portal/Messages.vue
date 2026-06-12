<template>
  <div class="messages-page">
    <h1 class="page-title">站内消息</h1>

    <div class="message-list" v-if="messages.length > 0">
      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['message-item', { 'message-item--unread': !msg.is_read }]"
        @click="handleClick(msg)"
      >
        <div class="message-dot" v-if="!msg.is_read"></div>
        <div class="message-body">
          <div class="message-header">
            <span class="message-title">{{ msg.title }}</span>
            <span class="message-time">{{ formatDate(msg.created_at) }}</span>
          </div>
          <p class="message-content">{{ msg.content }}</p>
        </div>
      </div>
    </div>

    <div v-else-if="!loading" class="empty-state">暂无消息</div>
    <div v-if="loading" class="loading-text">加载中...</div>

    <!-- 分页 -->
    <Pagination
      v-if="total > pageSize"
      :current-page="page"
      :total="total"
      :page-size="pageSize"
      @update:current-page="handlePageChange"
    />
  </div>
</template>

<script setup lang="ts">
// TODO(portal/Messages): Pagination 组件仅 emit update:current-page/update:page-size，
//                       但模板使用 @change="handlePageChange" 监听不存在的事件 — 分页功能实际失效。
//                       应改为 @update:current-page="handlePageChange" 或使用 v-model:current-page。
// TODO(portal/Messages): handleClick 中 markAsRead 无错误处理 — 标记失败时用户无感知。
// TODO(portal/Messages): 使用 (res as any) 绕过类型检查 — 待 API 层泛型补全后移除。
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listMessages, markAsRead, type MessageItem } from '@/api/message'
import Pagination from '@/components/common/Pagination.vue'

const router = useRouter()
const messages = ref<MessageItem[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

onMounted(() => {
  loadMessages()
})

async function loadMessages() {
  loading.value = true
  try {
    const res = await listMessages(page.value, pageSize.value)
    const data = (res as any).data || res
    messages.value = data?.items || []
    total.value = data?.total || 0
  } catch (err) {
    console.error('加载消息列表失败', err)
    messages.value = []
  } finally {
    loading.value = false
  }
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadMessages()
}

async function handleClick(msg: MessageItem) {
  // 标记已读
  if (!msg.is_read) {
    try {
      await markAsRead(msg.id)
      msg.is_read = true
    } catch (err) {
      console.error('标记消息已读失败', err)
    }
  }

  // 如果是申告相关消息，跳转到申告详情
  if (msg.related_type === 'ticket' && msg.related_id) {
    router.push(`/portal/tickets/${msg.related_id}`)
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}
</script>

<style scoped>
.page-title {
  font-size: 24px;
  font-weight: var(--font-weight-semibold, 600);
  margin-bottom: 24px;
}

.message-list {
  background: var(--bg-panel);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}

.message-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-default);
  cursor: pointer;
  transition: background 0.1s;
}

.message-item:last-child { border-bottom: none; }
.message-item:hover { background: var(--bg-overlay); }

.message-item--unread {
  background: rgba(94, 106, 210, 0.05);
}

.message-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  flex-shrink: 0;
  margin-top: 6px;
}

.message-body { flex: 1; min-width: 0; }

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.message-title {
  font-size: 14px;
  font-weight: 500;
}

.message-time {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
  margin-left: 16px;
}

.message-content {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.empty-state {
  text-align: center;
  padding: 64px;
  color: var(--text-secondary);
  font-size: 15px;
}

.loading-text {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
