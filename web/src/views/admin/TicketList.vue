<template>
  <div class="ticket-list-page">
    <div class="page-header"><h1 class="page-title">申告管理</h1></div>
    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead><tr>
          <th>编号</th><th>标题</th><th>报障人</th><th>紧急度</th><th>状态</th><th>创建时间</th><th>操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="ticket in tickets" :key="ticket.id">
            <td class="ticket-no">{{ ticket.ticket_no }}</td>
            <td>{{ ticket.title }}</td>
            <td>{{ ticket.submitter_name || '-' }}</td>
            <td><span :class="['urgency-tag', urgencyClass(ticket.urgency)]">{{ urgencyText(ticket.urgency) }}</span></td>
            <td><span :class="['status-tag', statusClass(ticket.status)]">{{ ticket.status_text }}</span></td>
            <td>{{ formatDate(ticket.created_at) }}</td>
            <td><button class="btn-action" @click="$router.push(`/admin/tickets/${ticket.id}`)">处理</button></td>
          </tr>
        </tbody>
      </table>
      <div v-if="tickets.length === 0" class="empty-state"><p>暂无申告</p></div>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/TicketList): catch 块静默忽略错误（/* API error */）— API 调用失败时用户无提示。
// TODO(admin/TicketList): urgencyClass/urgencyText/statusClass/formatDate 在 admin/TicketDetail、
//                       portal/TicketQuery、portal/TicketDetail 中重复 — 应抽取到 utils/ticket.ts。
// TODO(admin/TicketList): 使用 (res as any) 强制类型转换 — 等 API 泛型补全后移除。
import { ref, onMounted } from 'vue'
import { listAllTickets } from '@/api/admin'
import type { TicketItem } from '@/api/ticket'

const loading = ref(true); const tickets = ref<TicketItem[]>([])

onMounted(async () => {
  try { const res = await listAllTickets({ page_size: 50 }) as any; tickets.value = res?.data || res?.items || [] }
  catch { /* API error */ }
  finally { loading.value = false }
})

function urgencyClass(u: number) { return u === 3 ? 'high' : u === 2 ? 'medium' : 'low' }
function urgencyText(u: number) { return u === 3 ? '高' : u === 2 ? '中' : '低' }
function statusClass(s: number) {
  if (s === 1) return 'pending'; if (s === 2) return 'processing'; if (s === 3) return 'supplement'; if (s === 4) return 'resolved'; return 'closed'
}
function formatDate(d: string) { if (!d) return '-'; return d.substring(0, 10) }
</script>

<style scoped>
.ticket-list-page { max-width: 960px; }
.page-header { margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: 510; color: var(--text-primary); }
.loading-state, .empty-state { text-align: center; padding: 48px; color: var(--text-secondary); font-size: 14px; }
.table-wrapper { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 10px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 12px 16px; font-size: 12px; font-weight: 510; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid var(--border-default); background: var(--bg-base); }
.data-table td { padding: 12px 16px; border-bottom: 1px solid var(--border-default); font-size: 14px; color: var(--text-primary); }
.data-table tr:last-child td { border-bottom: none; }
.ticket-no { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; color: var(--text-secondary); }
.urgency-tag { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 12px; }
.urgency-tag.high { background: var(--tag-urgent-bg); color: var(--tag-urgent-text); }
.urgency-tag.medium { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.urgency-tag.low { background: rgba(94,106,210,0.12); color: var(--accent); }
.status-tag { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 12px; }
.status-tag.pending { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.status-tag.processing { background: var(--tag-processing-bg); color: var(--tag-processing-text); }
.status-tag.supplement { background: var(--tag-supplement-bg); color: var(--tag-supplement-text); }
.status-tag.resolved { background: var(--tag-resolved-bg); color: var(--tag-resolved-text); }
.status-tag.closed { background: var(--tag-disabled-bg); color: var(--tag-disabled-text); }
.btn-action { padding: 4px 14px; background: var(--accent); color: #fff; border: none; border-radius: 5px; font-size: 12px; cursor: pointer; font-family: inherit; }
.btn-action:hover { background: var(--accent-hover); }
</style>
