<template>
  <div class="role-manage-page">
    <div class="page-header">
      <h1 class="page-title">角色管理</h1>
      <button class="btn-primary" @click="openCreate">新建角色</button>
    </div>

    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead><tr>
          <th>ID</th><th>名称</th><th>描述</th><th>权限</th><th>创建时间</th><th>操作</th>
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
            <td class="actions">
              <button class="btn-action" @click="openEdit(role)">编辑</button>
              <button class="btn-action btn-danger" @click="handleDelete(role)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 创建/编辑对话框 -->
    <div v-if="showDialog" class="dialog-overlay" @click.self="closeDialog">
      <div class="dialog">
        <h2 class="dialog-title">{{ editingRole ? '编辑角色' : '新建角色' }}</h2>
        <div class="dialog-body">
          <label class="field-label">名称 <span class="required">*</span></label>
          <input v-model="form.name" class="field-input" placeholder="角色名称" />
          <label class="field-label">描述</label>
          <input v-model="form.description" class="field-input" placeholder="角色描述" />
          <label class="field-label">权限</label>
          <div class="permissions-grid">
            <label v-for="p in availablePermissions" :key="p" class="perm-checkbox">
              <input type="checkbox" :value="p" v-model="form.permissions" />
              {{ p }}
            </label>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-secondary" @click="closeDialog">取消</button>
          <button class="btn-primary" :disabled="saving" @click="handleSave">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getRoleList, createRole, updateRole, deleteRole } from '@/api/role'
import type { RoleItem } from '@/api/role'
import { useToast } from '@/composables/useToast'

const loading = ref(true)
const roles = ref<RoleItem[]>([])
const toast = useToast()

// 对话框状态
const showDialog = ref(false)
const editingRole = ref<RoleItem | null>(null)
const saving = ref(false)
const form = ref({ name: '', description: '', permissions: [] as string[] })

const availablePermissions = ['user:manage', 'ticket:read', 'ticket:write', 'knowledge:read', 'knowledge:write', 'knowledge:review', 'audit:read', 'system:config']

async function loadRoles() {
  loading.value = true
  try {
    const res = await getRoleList()
    // 后端返回分页结构 { data: { items: [...], total } }
    const data = (res as any).data
    roles.value = data?.items || data || []
  } catch (err) {
    console.error('加载角色列表失败', err)
    toast.showToast('加载角色列表失败', 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingRole.value = null
  form.value = { name: '', description: '', permissions: [] }
  showDialog.value = true
}

function openEdit(role: RoleItem) {
  editingRole.value = role
  form.value = { name: role.name, description: role.description, permissions: [...(role.permissions || [])] }
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editingRole.value = null
}

async function handleSave() {
  if (!form.value.name.trim()) { toast.showToast('角色名称不能为空', 'error'); return }
  saving.value = true
  try {
    if (editingRole.value) {
      await updateRole(editingRole.value.id, form.value)
      toast.showToast('角色已更新', 'success')
    } else {
      await createRole(form.value)
      toast.showToast('角色已创建', 'success')
    }
    closeDialog()
    await loadRoles()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as any)?.message || '保存失败'
    toast.showToast(msg, 'error')
  } finally {
    saving.value = false
  }
}

async function handleDelete(role: RoleItem) {
  if (!confirm(`确定要删除角色「${role.name}」吗？`)) return
  try {
    await deleteRole(role.id)
    toast.showToast('角色已删除', 'success')
    await loadRoles()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as any)?.message || '删除失败'
    toast.showToast(msg, 'error')
  }
}

onMounted(loadRoles)
</script>

<style scoped>
.role-manage-page { max-width: 960px; }
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 28px; }
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
.actions { display: flex; gap: 8px; }
.btn-action { padding: 4px 12px; border-radius: 6px; font-size: 12px; font-weight: 500; border: 1px solid var(--border-default); background: var(--bg-overlay); color: var(--text-primary); cursor: pointer; transition: all 0.15s; }
.btn-action:hover { background: var(--border-default); }
.btn-danger { color: var(--color-danger); border-color: rgba(229,72,77,0.3); }
.btn-danger:hover { background: rgba(229,72,77,0.12); }
.btn-primary { padding: 6px 16px; border-radius: 6px; font-size: 13px; font-weight: 500; background: var(--accent); color: #fff; border: none; cursor: pointer; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { padding: 6px 16px; border-radius: 6px; font-size: 13px; font-weight: 500; background: transparent; color: var(--text-secondary); border: 1px solid var(--border-default); cursor: pointer; }

/* 对话框样式 */
.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 12px; padding: 24px; width: 480px; max-height: 90vh; overflow-y: auto; }
.dialog-title { font-size: 17px; font-weight: 510; color: var(--text-primary); margin-bottom: 20px; }
.dialog-body { display: flex; flex-direction: column; gap: 12px; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 8px; margin-top: 24px; }
.field-label { font-size: 13px; font-weight: 500; color: var(--text-secondary); }
.required { color: var(--color-danger); }
.field-input { padding: 8px 12px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--bg-base); color: var(--text-primary); font-size: 13px; }
.field-input:focus { outline: none; border-color: var(--accent); }
.permissions-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.perm-checkbox { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--text-primary); cursor: pointer; }
</style>
