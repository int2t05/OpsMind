<template>
  <div class="ticket-detail-page">
    <div class="back-nav">
      <router-link to="/portal/tickets" class="back-link">&larr; 返回我的申告</router-link>
    </div>

    <div v-if="loading" class="loading-text">加载中...</div>

    <template v-if="ticket && !loading">
      <h1 class="page-title">{{ ticket.ticket_no }}</h1>

      <!-- 基本信息 -->
      <div class="card">
        <h2 class="card-title">{{ ticket.title }}</h2>
        <StatusBadge :status="ticket.status" type="ticket" />
        <p class="description">{{ ticket.description }}</p>

        <div class="meta-grid">
          <div class="meta-item">
            <span class="meta-label">紧急程度</span>
            <span>{{ urgencyText(ticket.urgency) }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">影响范围</span>
            <span>{{ impactText(ticket.impact_scope) }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">联系电话</span>
            <span>{{ ticket.contact_phone }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">联系邮箱</span>
            <span>{{ ticket.contact_email || '-' }}</span>
          </div>
          <div class="meta-item" v-if="ticket.affected_systems?.length">
            <span class="meta-label">受影响系统</span>
            <span>{{ ticket.affected_systems.join(', ') }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">创建时间</span>
            <span>{{ formatDate(ticket.created_at) }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">补充次数</span>
            <span>{{ ticket.supplement_count }} / 3</span>
          </div>
        </div>
      </div>

      <!-- 处理记录时间线 -->
      <div class="card" v-if="ticket.records?.length">
        <h3 class="section-title">处理记录</h3>
        <div class="timeline">
          <div v-for="record in ticket.records" :key="record.id" class="timeline-item">
            <div class="timeline-dot"></div>
            <div class="timeline-content">
              <div class="timeline-header">
                <span class="timeline-action">{{ actionText(record.action) }}</span>
                <span class="timeline-time">{{ formatDate(record.created_at) }}</span>
              </div>
              <p v-if="record.content" class="timeline-text">{{ record.content }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 补充信息入口（仅"需补充信息"状态） -->
      <div v-if="ticket.status === 3" class="card supplement-section">
        <h3 class="section-title">补充信息</h3>
        <p class="supplement-hint">运维人员需要您提供更多信息，请在此补充（剩余 {{ 3 - ticket.supplement_count }} 次机会）</p>
        <textarea
          v-model="supplementContent"
          class="form-textarea"
          rows="4"
          placeholder="请补充描述问题详情..."
        ></textarea>
        <button
          class="btn-primary"
          :disabled="submitting || !supplementContent.trim()"
          @click="handleSupplement"
        >
          {{ submitting ? '提交中...' : '提交补充信息' }}
        </button>
        <span v-if="supplementError" class="error-text">{{ supplementError }}</span>
        <span v-if="supplementSuccess" class="success-text">补充信息已提交</span>
      </div>
    </template>

    <div v-if="!loading && !ticket" class="empty-state">
      <p>申告不存在或无权查看</p>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(portal/TicketDetail): API 调用失败时静默置 null，无用户可见错误提示。
// TODO(portal/TicketDetail): 使用 (res as any) / (res?.data || res) 解包 — 等 API 泛型补全后统一。
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getTicketDetail, supplementTicket, type TicketDetail } from '@/api/ticket'
import { urgencyText, scopeText as impactText, actionText } from '@/utils/ticket'
import { formatDate } from '@/utils/date'
import StatusBadge from '@/components/common/StatusBadge.vue'

const route = useRoute()
const ticketId = Number(route.params.id)

const ticket = ref<TicketDetail | null>(null)
const loading = ref(true)
const supplementContent = ref('')
const submitting = ref(false)
const supplementError = ref('')
const supplementSuccess = ref(false)

onMounted(async () => {
  try {
    const res = await getTicketDetail(ticketId)
    ticket.value = ((res as any).data || res) as TicketDetail
  } catch {
    ticket.value = null
  } finally {
    loading.value = false
  }
})

async function handleSupplement() {
  if (!supplementContent.value.trim()) return
  supplementError.value = ''
  supplementSuccess.value = false
  submitting.value = true

  try {
    await supplementTicket(ticketId, { content: supplementContent.value.trim() })
    supplementSuccess.value = true
    supplementContent.value = ''
    // 刷新详情
    const res = await getTicketDetail(ticketId)
    ticket.value = ((res as any).data || res) as TicketDetail
  } catch {
    supplementError.value = '提交失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

// urgencyText/impactText/actionText → @/utils/ticket.ts / formatDate → @/utils/date.ts
</script>

<style scoped>
.page-title {
  font-size: 20px;
  font-weight: var(--font-weight-semibold, 600);
  margin-bottom: 24px;
  font-family: monospace;
  color: var(--accent);
}

.back-nav { margin-bottom: 16px; }
.back-link { color: var(--text-secondary); text-decoration: none; font-size: 13px; }
.back-link:hover { color: var(--text-primary); }

.card {
  background: var(--bg-panel);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 20px;
}

.card-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 12px;
}

.description {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
  margin: 16px 0;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 13px;
}

.meta-label {
  color: var(--text-secondary);
  font-size: 11px;
  text-transform: uppercase;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 16px;
}

/* 时间线 */
.timeline {
  position: relative;
  padding-left: 20px;
}

.timeline::before {
  content: '';
  position: absolute;
  left: 6px;
  top: 4px;
  bottom: 4px;
  width: 1px;
  background: var(--border-default);
}

.timeline-item {
  position: relative;
  margin-bottom: 20px;
}

.timeline-dot {
  position: absolute;
  left: -14px;
  top: 4px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--accent);
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.timeline-action {
  font-size: 13px;
  font-weight: 500;
}

.timeline-time {
  font-size: 11px;
  color: var(--text-secondary);
}

.timeline-text {
  color: var(--text-secondary);
  font-size: 13px;
}

/* 补充信息 */
.supplement-hint {
  color: var(--text-secondary);
  font-size: 13px;
  margin-bottom: 12px;
}

.form-textarea {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.btn-primary {
  display: inline-block;
  margin-top: 12px;
  padding: 10px 24px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
}

.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.error-text { color: var(--tag-rejected-text); font-size: 13px; margin-left: 12px; }
.success-text { color: var(--tag-published-text); font-size: 13px; margin-left: 12px; }

.loading-text { text-align: center; padding: 48px; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 64px; color: var(--text-secondary); }
</style>
