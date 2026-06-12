<template>
  <div class="role-manage-page">
    <div class="page-header"><h1 class="page-title">角色管理</h1></div>
    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead><tr>
          <th>ID</th><th>名称</th><th>描述</th><th>权限</th><th>创建时间</th>
        </tr></thead>
        <tbody>
          <tr v-for="role in roles" :key="role.id">
            <td>{{ role.id }}</td>
            <td class="role-name">{{ role.name }}</td>
            <td>{{ role.description }}</td>
            <td>
              <span v-for="p in role.permissions" :key="p" class="perm-tag">{{ p }}</span>
              <span v-if="!role.permissions || role.permissions.length === 0" class="text-muted">无权限</span>
            </td>
            <td>{{ role.created_at ? role.created_at.substring(0, 10) : '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/RoleManage): catch 块静默忽略所有错误 — API 调用失败时用户无任何提示。
// TODO(admin/RoleManage): 使用 (res as any) 强制类型转换 — 等 API 层泛型补全后移除。
import { ref, onMounted } from 'vue'
import { getRoleList } from '@/api/role'
import type { RoleItem } from '@/api/role'

const loading = ref(true); const roles = ref<RoleItem[]>([])

onMounted(async () => {
  try { const res = await getRoleList() as any; roles.value = res?.data || res || [] }
  catch (err) { console.error('加载角色列表失败', err) }
  finally { loading.value = false }
})
</script>

<style scoped>
.role-manage-page { max-width: 900px; }
.page-header { margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: 510; color: var(--text-primary); }
.loading-state { text-align: center; padding: 48px; color: var(--text-secondary); font-size: 14px; }
.table-wrapper { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 10px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 12px 16px; font-size: 12px; font-weight: 510; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid var(--border-default); background: var(--bg-base); }
.data-table td { padding: 12px 16px; border-bottom: 1px solid var(--border-default); font-size: 14px; color: var(--text-primary); }
.data-table tr:last-child td { border-bottom: none; }
.role-name { font-weight: 510; }
.perm-tag { display: inline-block; padding: 1px 8px; margin: 2px 4px 2px 0; background: rgba(94,106,210,0.1); color: var(--accent); border-radius: 4px; font-size: 11px; font-family: 'SF Mono', 'Fira Code', monospace; }
.text-muted { color: var(--text-secondary); font-size: 13px; }
</style>
