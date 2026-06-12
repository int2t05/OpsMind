<template>
  <div class="ticket-query-page">
    <h1 class="page-title">我的申告</h1>

    <div class="table-container">
      <table class="data-table" v-if="tickets.length > 0">
        <thead>
          <tr>
            <th>编号</th>
            <th>标题</th>
            <th>状态</th>
            <th>紧急程度</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ticket in tickets" :key="ticket.id">
            <td class="ticket-no">{{ ticket.ticket_no }}</td>
            <td>{{ ticket.title }}</td>
            <td><StatusBadge :status="ticket.status" type="ticket" /></td>
            <td>{{ urgencyText(ticket.urgency) }}</td>
            <td class="text-secondary">{{ formatDate(ticket.created_at) }}</td>
            <td>
              <router-link :to="`/portal/tickets/${ticket.id}`" class="link">查看</router-link>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-else-if="!loading" class="empty-state">
        <p>暂无申告记录</p>
        <router-link to="/portal/tickets/submit" class="btn-primary">提交申告</router-link>
      </div>

      <div v-if="loading" class="loading-text">加载中...</div>
    </div>

    <!-- 分页 -->
    <Pagination
      v-if="total > pageSize"
      :current-page="page"
      :total="total"
      :page-size="pageSize"
      @change="handlePageChange"
    />
  </div>
</template>

<script setup lang="ts">
// TODO(portal/TicketQuery): Pagination 组件 @change 事件不存在 — 应改为 @update:current-page 或 v-model:current-page。
// TODO(portal/TicketQuery): urgencyText 函数与 admin/TicketDetail、admin/TicketList、portal/TicketDetail 重复 —
//                          应提取到 utils/ticket.ts 作为共享工具函数。
// TODO(portal/TicketQuery): formatDate 函数在几乎所有页面中重复定义 — 应抽取为共享工具函数或全局 filter。
import { ref, onMounted } from 'vue'
import { listMyTickets, type TicketItem } from '@/api/ticket'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Pagination from '@/components/common/Pagination.vue'

const tickets = ref<TicketItem[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

onMounted(() => {
  loadTickets()
})

async function loadTickets() {
  loading.value = true
  try {
    const res = await listMyTickets(page.value, pageSize.value)
    const data = (res as any).data || res
    tickets.value = data?.items || []
    total.value = data?.total || 0
  } catch {
    tickets.value = []
  } finally {
    loading.value = false
  }
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadTickets()
}

function urgencyText(urgency: number): string {
  const map: Record<number, string> = { 1: '低', 2: '中', 3: '高' }
  return map[urgency] || '未知'
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

.table-container {
  background: var(--bg-panel);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th,
.data-table td {
  padding: 12px 16px;
  text-align: left;
  font-size: 13px;
  border-bottom: 1px solid var(--border-default);
}

.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  background: var(--bg-overlay);
  font-size: 12px;
  text-transform: uppercase;
}

.data-table td { color: var(--text-primary); }

.ticket-no {
  font-family: monospace;
  font-size: 12px;
  color: var(--accent);
}

.text-secondary { color: var(--text-secondary); font-size: 12px; }

.link {
  color: var(--accent);
  text-decoration: none;
  font-size: 13px;
}

.link:hover { text-decoration: underline; }

.empty-state {
  text-align: center;
  padding: 64px 24px;
  color: var(--text-secondary);
}

.empty-state p {
  margin-bottom: 16px;
  font-size: 15px;
}

.btn-primary {
  display: inline-block;
  padding: 10px 24px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  text-decoration: none;
  cursor: pointer;
}

.btn-primary:hover { background: var(--accent-hover); }

.loading-text {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
