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
            <td>{{ log.ip || '-' }}</td>
            <td class="time-cell">{{ formatTime(log.created_at) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="logs.length === 0" class="empty-state"><p>暂无审计日志</p></div>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/AuditLog): catch 块静默忽略所有错误（/* ignore */），API 调用失败时用户无任何提示。
//                      应展示错误提示并记录到 console.error。
// TODO(admin/AuditLog): 使用 (res as any) 绕过类型检查 — 应使用审计日志 API 模块的泛型声明。
// TODO(admin/AuditLog): page 和 page_size 硬编码 — 应支持分页参数。
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

interface AuditLogItem {
  id: number; username?: string; operator_name?: string; action: string
  target_type?: string; target_id?: number; detail?: string; ip?: string; created_at?: string
}

const loading = ref(true); const logs = ref<AuditLogItem[]>([])

onMounted(async () => {
  try {
    const res = await request.get('/api/v1/admin/audit-logs', { params: { page: 1, page_size: 100 } }) as any
    logs.value = res?.data || res?.items || []
  } catch (err) { console.error('加载审计日志失败', err) }
  finally { loading.value = false }
})

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
