<template>
  <div class="ticket-detail-page">
    <div class="page-header">
      <button class="btn-back" @click="$router.push('/admin/tickets')">← 返回列表</button>
      <h1 class="page-title">申告处理</h1>
    </div>

    <div v-if="loading" class="loading-state"><p>加载中...</p></div>
    <div v-else-if="ticket" class="detail-content">
      <!-- 基本信息 -->
      <div class="card">
        <div class="card-header">
          <div>
            <span class="ticket-no">{{ ticket.ticket_no }}</span>
            <h2>{{ ticket.title }}</h2>
          </div>
          <span :class="['status-tag', statusClass(ticket.status)]">{{ ticket.status_text }}</span>
        </div>
        <div class="info-grid">
          <div class="info-item"><span class="label">报障人</span><span>{{ ticket.submitter_name || '-' }}</span></div>
          <div class="info-item"><span class="label">联系电话</span><span>{{ ticket.contact_phone }}</span></div>
          <div class="info-item"><span class="label">邮箱</span><span>{{ ticket.contact_email || '-' }}</span></div>
          <div class="info-item"><span class="label">紧急度</span><span>{{ urgencyText(ticket.urgency) }}</span></div>
          <div class="info-item"><span class="label">影响范围</span><span>{{ scopeText(ticket.impact_scope) }}</span></div>
          <div class="info-item"><span class="label">补充次数</span><span>{{ ticket.supplement_count }}/3</span></div>
          <div class="info-item"><span class="label">创建时间</span><span>{{ ticket.created_at }}</span></div>
        </div>
        <div class="description">
          <span class="label">问题描述</span>
          <p>{{ ticket.description }}</p>
        </div>
      </div>

      <!-- 状态操作 -->
      <div class="card" v-if="ticket.status !== 4 && ticket.status !== 5">
        <h3>状态操作</h3>
        <div class="action-bar">
          <template v-if="ticket.status === 1">
            <button class="btn-primary" @click="doAction('start')">接单</button>
          </template>
          <template v-if="ticket.status === 2">
            <button class="btn-primary" @click="doAction('resolve')">已解决</button>
            <button class="btn-warn" @click="doAction('request_info')">需补充信息</button>
          </template>
          <template v-if="ticket.status === 3">
            <button class="btn-primary" @click="doAction('resolve')">已解决</button>
          </template>
          <textarea v-model="actionContent" class="action-textarea" placeholder="处理备注..." rows="2" />
        </div>
      </div>

      <!-- 添加处理记录 -->
      <div class="card">
        <h3>添加处理记录</h3>
        <div class="record-form">
          <select v-model="recordAction" class="field-select">
            <option value="note">备注</option>
            <option value="visit">回访</option>
            <option value="escalate">升级</option>
          </select>
          <textarea v-model="recordContent" class="action-textarea" placeholder="记录内容..." rows="2" />
          <button class="btn-primary" :disabled="savingRecord" @click="doAddRecord">
            {{ savingRecord ? '保存中...' : '添加记录' }}
          </button>
        </div>
      </div>

      <!-- 知识候选 -->
      <div class="card">
        <h3>转为知识候选</h3>
        <div class="record-form">
          <label class="kb-label">目标知识库 ID</label>
          <input v-model.number="candidateKBID" type="number" class="field-input" placeholder="输入知识库 ID" style="width:120px" />
          <button class="btn-primary" :disabled="!candidateKBID || creatingCandidate" @click="handleCreateCandidate">
            {{ creatingCandidate ? '创建中...' : '创建知识候选' }}
          </button>
        </div>
      </div>

      <!-- 处理记录 -->
      <div class="card">
        <h3>处理记录</h3>
        <div v-if="ticket.records && ticket.records.length > 0" class="records">
          <div v-for="rec in ticket.records" :key="rec.id" class="record-item">
            <div class="record-header">
              <span :class="['record-action', rec.action]">{{ actionText(rec.action) }}</span>
              <span class="record-time">{{ rec.created_at }}</span>
            </div>
            <p v-if="rec.content" class="record-content">{{ rec.content }}</p>
          </div>
        </div>
        <div v-else class="empty-hint"><p>暂无处理记录</p></div>
      </div>
    </div>

    <div v-if="toast.visible.value" :class="['toast', toast.type.value]">{{ toast.message.value }}</div>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/TicketDetail): 使用 (res as any) 强制类型转换 — 等 API 泛型补全后移除。
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getTicketDetail, updateTicketStatus, addTicketRecord, createKnowledgeCandidate } from '@/api/admin'
import type { TicketDetail } from '@/api/ticket'
import { urgencyText, ticketStatusClass as statusClass, scopeText, actionText } from '@/utils/ticket'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const loading = ref(true); const saving = ref(false); const savingRecord = ref(false)
const ticket = ref<TicketDetail | null>(null)
const actionContent = ref('')
const recordAction = ref('note')
const recordContent = ref('')
const candidateKBID = ref<number | null>(null)
const creatingCandidate = ref(false)
const toast = useToast()

onMounted(async () => {
  const id = Number(route.params.id)
  try { const res = await getTicketDetail(id) as any; ticket.value = res?.data || res }
  catch (err) { console.error('加载申告失败', err); toast.showToast('加载申告失败', 'error') }
  finally { loading.value = false }
})

// toast 已通过 useToast composable 管理，自动处理定时器清理

async function doAction(action: string) {
  saving.value = true
  try {
    await updateTicketStatus(ticket.value!.id, { action, result: actionContent.value })
    toast.showToast('操作成功', 'success')
    actionContent.value = ''
    // Reload detail
    const res = await getTicketDetail(ticket.value!.id) as any; ticket.value = res?.data || res
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : '操作失败'; toast.showToast(msg, 'error') }
  finally { saving.value = false }
}

async function doAddRecord() {
  if (!recordContent.value.trim()) { toast.showToast('记录内容不能为空', 'error'); return }
  savingRecord.value = true
  try {
    await addTicketRecord(ticket.value!.id, { action: recordAction.value, content: recordContent.value })
    toast.showToast('记录已添加', 'success')
    recordContent.value = ''
    // Reload detail to show new record
    const res = await getTicketDetail(ticket.value!.id) as any; ticket.value = res?.data || res
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : '操作失败'; toast.showToast(msg, 'error') }
  finally { savingRecord.value = false }
}

async function handleCreateCandidate() {
  if (!candidateKBID.value || !ticket.value) return
  creatingCandidate.value = true
  try {
    await createKnowledgeCandidate(ticket.value.id, candidateKBID.value)
    toast.showToast('知识候选创建成功', 'success')
    candidateKBID.value = null
  } catch (e: any) {
    toast.showToast(e?.response?.data?.message || e?.message || '创建失败', 'error')
  } finally { creatingCandidate.value = false }
}

// statusClass/urgencyText/scopeText/actionText → @/utils/ticket.ts
</script>

<style scoped>
.ticket-detail-page { max-width: 800px; }
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: 510; color: var(--text-primary); }
.btn-back { padding: 6px 14px; background: var(--bg-base); color: var(--text-secondary); border: 1px solid var(--border-default); border-radius: 6px; font-size: 13px; cursor: pointer; font-family: inherit; }
.btn-back:hover { color: var(--text-primary); }
.loading-state { text-align: center; padding: 48px; color: var(--text-secondary); }
.card { background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 10px; padding: 24px; margin-bottom: 20px; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 18px; }
.ticket-no { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; color: var(--text-secondary); }
.card h2 { font-size: 18px; font-weight: 510; color: var(--text-primary); margin: 4px 0 0; }
.card h3 { font-size: 14px; font-weight: 510; color: var(--text-secondary); margin: 0 0 16px; }
.info-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; margin-bottom: 18px; }
.info-item { display: flex; flex-direction: column; gap: 2px; }
.info-item .label { font-size: 11px; color: var(--text-secondary); text-transform: uppercase; }
.info-item span:last-child { font-size: 14px; color: var(--text-primary); }
.description { border-top: 1px solid var(--border-default); padding-top: 14px; }
.description .label { font-size: 11px; color: var(--text-secondary); display: block; margin-bottom: 6px; }
.description p { font-size: 14px; color: var(--text-primary); line-height: 1.6; margin: 0; }
.status-tag { display: inline-block; padding: 2px 12px; border-radius: 10px; font-size: 13px; }
.status-tag.pending { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.status-tag.processing { background: var(--tag-processing-bg); color: var(--tag-processing-text); }
.status-tag.supplement { background: var(--tag-supplement-bg); color: var(--tag-supplement-text); }
.status-tag.resolved { background: var(--tag-resolved-bg); color: var(--tag-resolved-text); }
.status-tag.closed { background: var(--tag-disabled-bg); color: var(--tag-disabled-text); }
.action-bar { display: flex; flex-wrap: wrap; gap: 8px; align-items: flex-start; }
.btn-primary { padding: 8px 20px; background: var(--accent); color: #fff; border: none; border-radius: 8px; font-size: 13px; cursor: pointer; font-family: inherit; }
.btn-primary:hover { background: var(--accent-hover); }
.btn-warn { padding: 8px 20px; background: var(--btn-warning-bg); color: var(--btn-warning-text); border: 1px solid var(--border-default); border-radius: 8px; font-size: 13px; cursor: pointer; font-family: inherit; }
.action-textarea { flex-basis: 100%; padding: 10px; background: var(--bg-base); border: 1px solid var(--border-default); border-radius: 6px; color: var(--text-primary); font-size: 13px; font-family: inherit; resize: vertical; }
.action-textarea:focus { outline: none; border-color: var(--accent); }
.records { border-top: 1px solid var(--border-default); padding-top: 8px; }
.record-item { padding: 12px 0; border-bottom: 1px solid var(--border-default); }
.record-item:last-child { border-bottom: none; }
.record-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.record-action { display: inline-block; padding: 1px 8px; border-radius: 4px; font-size: 11px; font-weight: 510; }
.record-action.start { background: var(--tag-processing-bg); color: var(--tag-processing-text); }
.record-action.resolve { background: var(--tag-resolved-bg); color: var(--tag-resolved-text); }
.record-action.request_info { background: var(--tag-supplement-bg); color: var(--tag-supplement-text); }
.record-action.remark { background: var(--tag-disabled-bg); color: var(--tag-disabled-text); }
.record-action.supplement { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.record-time { font-size: 11px; color: var(--text-secondary); }
.record-content { font-size: 14px; color: var(--text-primary); margin: 4px 0 0; line-height: 1.5; }
.empty-hint { text-align: center; padding: 20px; color: var(--text-secondary); font-size: 13px; }
.record-form { display: flex; gap: 8px; align-items: flex-start; flex-wrap: wrap; }
.field-select { padding: 8px 12px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--bg-base); color: var(--text-primary); font-size: 13px; font-family: inherit; }
.field-select:focus { outline: none; border-color: var(--accent); }
.toast { position: fixed; bottom: 32px; right: 32px; padding: 12px 24px; border-radius: 8px; font-size: 14px; z-index: 9999; }
.toast.success { background: var(--toast-success-bg); color: var(--toast-success-text); border: 1px solid var(--toast-success-border); }
.toast.error { background: var(--toast-error-bg); color: var(--toast-error-text); border: 1px solid var(--toast-error-border); }
</style>
