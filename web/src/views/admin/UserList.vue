<template>
  <div class="user-list-page">
    <div class="page-header">
      <h1 class="page-title">用户管理</h1>
      <button class="btn-add" @click="startCreate">新增用户</button>
    </div>
    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead><tr>
          <th>ID</th><th>用户名</th><th>姓名</th><th>手机号</th><th>邮箱</th><th>状态</th><th>角色</th><th>操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.id }}</td>
            <td class="username">{{ user.username }}</td>
            <td>{{ user.real_name }}</td>
            <td>{{ user.phone }}</td>
            <td>{{ user.email || '-' }}</td>
            <td><span :class="['status-tag', user.status === 1 ? 'active' : 'frozen']">{{ user.status === 1 ? '正常' : '已冻结' }}</span></td>
            <td>{{ (user.roles || []).join('、') }}</td>
            <td class="action-cell">
              <button class="btn-action" @click="startEdit(user)">编辑</button>
              <button v-if="user.status === 1" class="btn-action danger" @click="handleFreeze(user.id)">冻结</button>
              <button v-else class="btn-action" @click="handleRestore(user.id)">恢复</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="users.length === 0" class="empty-state"><p>暂无用户数据</p></div>
    </div>

    <!-- 创建/编辑对话框 -->
    <div v-if="showDialog" class="dialog-overlay" @click.self="closeDialog">
      <div class="dialog">
        <h2>{{ isEdit ? '编辑用户' : '新增用户' }}</h2>
        <div class="form-group">
          <label>用户名 <span class="required">*</span></label>
          <input v-model="form.username" class="form-input" :disabled="isEdit" placeholder="字母开头，3-32位" />
        </div>
        <div class="form-group" v-if="!isEdit">
          <label>密码 <span class="required">*</span></label>
          <input v-model="form.password" class="form-input" type="password" placeholder="至少8位含大小写字母数字" />
        </div>
        <div class="form-group">
          <label>姓名 <span class="required">*</span></label>
          <input v-model="form.real_name" class="form-input" placeholder="真实姓名" />
        </div>
        <div class="form-group">
          <label>手机号 <span class="required">*</span></label>
          <input v-model="form.phone" class="form-input" placeholder="手机号" />
        </div>
        <div class="form-group">
          <label>邮箱</label>
          <input v-model="form.email" class="form-input" placeholder="选填" />
        </div>
        <div class="form-group">
          <label>角色 <span class="required">*</span></label>
          <select v-model="selectedRoles" class="form-select" multiple>
            <option v-for="r in allRoles" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </div>
        <div class="dialog-actions">
          <button class="btn-cancel" @click="closeDialog">取消</button>
          <button class="btn-save" :disabled="saving" @click="handleSave">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="toast.message" :class="['toast', toast.type]">{{ toast.message }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getUserList, createUser, updateUser, freezeUser, restoreUser } from '@/api/user'
import { getRoleList } from '@/api/role'
import type { RoleItem } from '@/api/role'

interface UserItem { id: number; username: string; real_name: string; phone: string; email: string; status: number; roles: string[]; role_ids?: number[] }

const loading = ref(true); const saving = ref(false)
const users = ref<UserItem[]>([]); const allRoles = ref<RoleItem[]>([])
const showDialog = ref(false); const isEdit = ref(false); const editingId = ref<number | null>(null)
const selectedRoles = ref<number[]>([])
const form = ref({ username: '', password: '', real_name: '', phone: '', email: '' })
const toast = ref<{ message: string; type: string }>({ message: '', type: 'success' })
let toastTimer: number | null = null

onMounted(async () => { await Promise.all([fetchUsers(), fetchRoles()]) })

function showToast(message: string, type: 'success' | 'error') {
  toast.value = { message, type }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => { toast.value = { message: '', type: 'success' } }, 3000)
}

async function fetchUsers() {
  loading.value = true
  try {
    const res = await getUserList({ page: 1, page_size: 100 }) as any
    users.value = res?.items || res?.data?.items || res || []
  } catch { showToast('加载用户失败', 'error') } finally { loading.value = false }
}

async function fetchRoles() {
  try { const res = await getRoleList() as any; allRoles.value = res?.data || res || [] } catch { /* ignore */ }
}

function startCreate() {
  form.value = { username: '', password: '', real_name: '', phone: '', email: '' }; selectedRoles.value = []
  isEdit.value = false; editingId.value = null; showDialog.value = true
}

function startEdit(user: UserItem) {
  form.value = { username: user.username, password: '', real_name: user.real_name, phone: user.phone, email: user.email || '' }
  selectedRoles.value = user.role_ids || []
  isEdit.value = true; editingId.value = user.id; showDialog.value = true
}

function closeDialog() { showDialog.value = false }

async function handleSave() {
  saving.value = true
  try {
    if (isEdit.value && editingId.value) {
      await updateUser(editingId.value, { real_name: form.value.real_name, phone: form.value.phone, email: form.value.email, role_ids: selectedRoles.value })
      showToast('更新成功', 'success')
    } else {
      await createUser({ ...form.value, role_ids: selectedRoles.value })
      showToast('创建成功', 'success')
    }
    closeDialog(); await fetchUsers()
  } catch (e: any) { showToast(e?.response?.data?.message || e?.message || '操作失败', 'error') }
  finally { saving.value = false }
}

async function handleFreeze(id: number) {
  try { await freezeUser(id); showToast('已冻结', 'success'); await fetchUsers() }
  catch (e: any) { showToast(e?.response?.data?.message || '冻结失败', 'error') }
}

async function handleRestore(id: number) {
  try { await restoreUser(id); showToast('已恢复', 'success'); await fetchUsers() }
  catch (e: any) { showToast(e?.response?.data?.message || '恢复失败', 'error') }
}
</script>

<style scoped>
.user-list-page { max-width: 960px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: 510; color: var(--text-primary); }
.btn-add { padding: 8px 18px; background: var(--accent); color: #fff; border: none; border-radius: 8px; font-size: 13px; cursor: pointer; font-family: inherit; }
.btn-add:hover { background: var(--accent-hover); }
.loading-state, .empty-state { text-align: center; padding: 48px; color: var(--text-secondary); font-size: 14px; }
.table-wrapper { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 10px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 12px 16px; font-size: 12px; font-weight: 510; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid var(--border-default); background: var(--bg-base); }
.data-table td { padding: 12px 16px; border-bottom: 1px solid var(--border-default); font-size: 14px; color: var(--text-primary); }
.data-table tr:last-child td { border-bottom: none; }
.username { font-weight: 510; }
.status-tag { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 12px; }
.status-tag.active { background: rgba(94,106,210,0.15); color: var(--accent); }
.status-tag.frozen { background: var(--tag-frozen-bg); color: var(--tag-frozen-text); }
.action-cell { text-align: right; white-space: nowrap; }
.btn-action { padding: 4px 10px; background: var(--bg-base); color: var(--text-secondary); border: 1px solid var(--border-default); border-radius: 5px; font-size: 12px; cursor: pointer; font-family: inherit; margin-left: 4px; }
.btn-action:hover { color: var(--text-primary); border-color: var(--border-hover); }
.btn-action.danger { color: var(--btn-danger-text); } .btn-action.danger:hover { background: var(--btn-danger-bg); }
.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.dialog { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 12px; padding: 28px; width: 420px; max-height: 80vh; overflow-y: auto; }
.dialog h2 { font-size: 18px; font-weight: 510; margin: 0 0 20px; color: var(--text-primary); }
.form-group { margin-bottom: 14px; }
.form-group label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 5px; }
.required { color: var(--tag-rejected-text); }
.form-input, .form-select { width: 100%; padding: 8px 12px; background: var(--bg-base); border: 1px solid var(--border-default); border-radius: 6px; color: var(--text-primary); font-size: 14px; font-family: inherit; box-sizing: border-box; }
.form-input:focus, .form-select:focus { outline: none; border-color: var(--accent); }
.form-input:disabled { opacity: 0.5; }
.form-select[multiple] { height: 100px; }
.dialog-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
.btn-cancel { padding: 8px 18px; background: var(--bg-base); color: var(--text-secondary); border: 1px solid var(--border-default); border-radius: 8px; font-size: 13px; cursor: pointer; font-family: inherit; }
.btn-save { padding: 8px 18px; background: var(--accent); color: #fff; border: none; border-radius: 8px; font-size: 13px; cursor: pointer; font-family: inherit; }
.btn-save:hover { background: var(--accent-hover); } .btn-save:disabled { opacity: 0.5; cursor: not-allowed; }
.toast { position: fixed; bottom: 32px; right: 32px; padding: 12px 24px; border-radius: 8px; font-size: 14px; z-index: 9999; }
.toast.success { background: var(--toast-success-bg); color: var(--toast-success-text); border: 1px solid var(--toast-success-border); }
.toast.error { background: var(--toast-error-bg); color: var(--toast-error-text); border: 1px solid var(--toast-error-border); }
</style>
