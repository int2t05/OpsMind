<template>
  <div class="audit-log-page">
    <div class="page-header"><h1 class="page-title">审计日志</h1></div>
    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead><tr>
          <th>ID</th><th>操作人</th><th>操作类型</th><th>目标</th><th>详情</th><th>IP</th><th>时间</th>
        </tr></thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id">
            <td>{{ log.id }}</td>
            <td>{{ log.username || log.operator_name || '-' }}</td>
            <td>{{ log.action }}</td>
            <td>{{ log.target_type }}{{ log.target_id ? '#' + log.target_id : '' }}</td>
            <td class="detail-cell">{{ log.detail || '-' }}</td>
            <td>{{ log.ip_address || '-' }}</td>
            <td class="time-cell">{{ formatTime(log.created_at) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="logs.length === 0" class="empty-state"><p>暂无审计日志</p></div>
    </div>
    <Pagination v-if="total > 0" :total="total" v-model:current-page="currentPage" v-model:page-size="pageSize" @update:current-page="fetchLogs" @update:page-size="fetchLogs" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listAuditLogs, type AuditLogItem } from '@/api/audit'
import { useToast } from '@/composables/useToast'
import Pagination from '@/components/common/Pagination.vue'

const loading = ref(true); const logs = ref<AuditLogItem[]>([])
const currentPage = ref(1); const pageSize = ref(20); const total = ref(0)
const toast = useToast()

async function fetchLogs() {
  loading.value = true
  try {
    const res = await listAuditLogs({ page: currentPage.value, page_size: pageSize.value })
    logs.value = (res as any).items || res.data?.items || []
    total.value = (res as any).total || logs.value.length
  } catch (err) {
    console.error('加载审计日志失败', err)
    toast.showToast('加载审计日志失败', 'error')
  } finally { loading.value = false }
}

onMounted(() => { fetchLogs() })

function formatTime(t?: string) { if (!t) return '-'; return t.substring(0, 19).replace('T', ' ') }
</script>

<style scoped>
.audit-log-page { max-width: 1000px; }
.page-header { margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: 510; color: var(--text-primary); }
.loading-state, .empty-state { text-align: center; padding: 48px; color: var(--text-secondary); font-size: 14px; }
.table-wrapper { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 10px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 12px 16px; font-size: 12px; font-weight: 510; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid var(--border-default); background: var(--bg-base); }
.data-table td { padding: 12px 16px; border-bottom: 1px solid var(--border-default); font-size: 14px; color: var(--text-primary); }
.data-table tr:last-child td { border-bottom: none; }
.detail-cell { max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.time-cell { font-size: 12px; color: var(--text-secondary); white-space: nowrap; }
</style>
