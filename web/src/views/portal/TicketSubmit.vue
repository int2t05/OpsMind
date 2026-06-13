<template>
  <div class="ticket-submit-page">
    <h1 class="page-title">提交申告</h1>

    <form class="submit-form" @submit.prevent="handleSubmit">
      <!-- 标题 -->
      <div class="form-group">
        <label class="form-label">标题 <span class="required">*</span></label>
        <input
          v-model="form.title"
          type="text"
          class="form-input"
          placeholder="简要描述问题"
          maxlength="255"
          required
        />
      </div>

      <!-- 描述 -->
      <div class="form-group">
        <label class="form-label">详细描述 <span class="required">*</span></label>
        <textarea
          v-model="form.description"
          class="form-textarea"
          rows="6"
          placeholder="请详细描述遇到的问题，包括时间、现象、影响范围等"
          required
        ></textarea>
      </div>

      <!-- 紧急程度 & 影响范围 -->
      <div class="form-row">
        <div class="form-group flex-1">
          <label class="form-label">紧急程度 <span class="required">*</span></label>
          <div class="radio-group">
            <label v-for="opt in urgencyOptions" :key="opt.value" class="radio-item">
              <input
                v-model="form.urgency"
                type="radio"
                :value="opt.value"
                required
              />
              <span>{{ opt.label }}</span>
            </label>
          </div>
        </div>
        <div class="form-group flex-1">
          <label class="form-label">影响范围</label>
          <div class="radio-group">
            <label v-for="opt in impactOptions" :key="opt.value" class="radio-item">
              <input
                v-model="form.impact_scope"
                type="radio"
                :value="opt.value"
              />
              <span>{{ opt.label }}</span>
            </label>
          </div>
        </div>
      </div>

      <!-- 受影响系统 -->
      <div class="form-group">
        <label class="form-label">受影响系统</label>
        <div class="tag-input">
          <span v-for="(tag, i) in form.affected_systems" :key="i" class="tag">
            {{ tag }}
            <button type="button" class="tag-remove" @click="removeTag(i)">&times;</button>
          </span>
          <input
            v-model="tagInput"
            type="text"
            class="tag-input-field"
            placeholder="输入系统名后按回车添加"
            @keydown.enter.prevent="addTag"
          />
        </div>
      </div>

      <!-- 联系方式 -->
      <div class="form-row">
        <div class="form-group flex-1">
          <label class="form-label">联系电话 <span class="required">*</span></label>
          <input
            v-model="form.contact_phone"
            type="tel"
            class="form-input"
            placeholder="请填写可联系到的手机号"
            maxlength="11"
            required
          />
        </div>
        <div class="form-group flex-1">
          <label class="form-label">联系邮箱</label>
          <input
            v-model="form.contact_email"
            type="email"
            class="form-input"
            placeholder="选填"
          />
        </div>
      </div>

      <!-- 提交 -->
      <div class="form-actions">
        <button type="submit" class="btn-primary" :disabled="submitting">
          {{ submitting ? '提交中...' : '提交申告' }}
        </button>
        <span v-if="submitError" class="error-text">{{ submitError }}</span>
        <span v-if="submitSuccess" class="success-text">申告提交成功！</span>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
// TODO(portal/TicketSubmit): 组件超过 340 行 — 可提取表单字段组件和验证逻辑。
import { ref, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createTicket } from '@/api/ticket'

const route = useRoute()
const router = useRouter()

const urgencyOptions = [
  { value: 1, label: '低' },
  { value: 2, label: '中' },
  { value: 3, label: '高' },
]

const impactOptions = [
  { value: 1, label: '个人' },
  { value: 2, label: '部门' },
  { value: 3, label: '全公司' },
]

const form = reactive({
  title: '',
  description: '',
  urgency: 2,
  impact_scope: 1,
  affected_systems: [] as string[],
  contact_phone: '',
  contact_email: '',
  // TODO(portal/TicketSubmit): chat_context 来自 URL query 参数直接拼接，无 JSON 解析校验。
  // 攻击者可通过 ?chat_context={"malicious":true} 注入任意值。应先 JSON.parse 并校验结构再使用。
  chat_context: (route.query.chat_context as string) || '',
})

const tagInput = ref('')
const submitting = ref(false)
const submitError = ref('')
const submitSuccess = ref(false)

function addTag() {
  const value = tagInput.value.trim()
  if (value && !form.affected_systems.includes(value)) {
    form.affected_systems.push(value)
  }
  tagInput.value = ''
}

function removeTag(index: number) {
  form.affected_systems.splice(index, 1)
}

async function handleSubmit() {
  submitError.value = ''
  submitSuccess.value = false

  if (!form.title.trim() || !form.description.trim() || !form.contact_phone.trim()) {
    submitError.value = '请填写标题、描述和联系电话'
    return
  }
  // 手机号格式校验（中国大陆手机号）
  if (!/^1\d{10}$/.test(form.contact_phone.trim())) {
    submitError.value = '请输入正确的手机号（11位，以1开头）'
    return
  }
  // 邮箱格式校验（可选字段，填写时校验）
  if (form.contact_email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.contact_email.trim())) {
    submitError.value = '请输入正确的邮箱地址'
    return
  }

  submitting.value = true
  try {
    await createTicket({
      title: form.title.trim(),
      description: form.description.trim(),
      urgency: form.urgency,
      impact_scope: form.impact_scope,
      affected_systems: form.affected_systems.filter(s => s.trim()).length > 0 ? form.affected_systems.filter(s => s.trim()) : undefined,
      contact_phone: form.contact_phone.trim(),
      contact_email: form.contact_email.trim() || undefined,
      chat_context: form.chat_context || undefined,
    })
    submitSuccess.value = true
    // 1.5 秒后跳转到我的申告列表
    setTimeout(() => router.push('/portal/tickets'), 1500)
  } catch (e: any) {
    submitError.value = e?.message || '提交失败，请稍后重试'
    submitting.value = false
  }
}
</script>

<style scoped>
.page-title {
  font-size: 24px;
  font-weight: var(--font-weight-semibold, 600);
  margin-bottom: 32px;
}

.submit-form {
  max-width: 720px;
}

.form-group {
  margin-bottom: 24px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.required { color: var(--accent-interactive); }

.form-input, .form-textarea {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  transition: border-color 0.15s;
}

.form-input:focus, .form-textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.form-textarea { resize: vertical; }

.form-row {
  display: flex;
  gap: 24px;
}

.flex-1 { flex: 1; }

.radio-group {
  display: flex;
  gap: 16px;
}

.radio-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  cursor: pointer;
  color: var(--text-secondary);
}

.radio-item input[type="radio"] {
  accent-color: var(--accent);
}

.radio-item:has(input:checked) {
  color: var(--text-primary);
}

/* 标签输入 */
.tag-input {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 10px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  min-height: 42px;
  align-items: center;
}

.tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: var(--accent);
  color: #fff;
  border-radius: 4px;
  font-size: 12px;
}

.tag-remove {
  background: none;
  border: none;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  line-height: 1;
  padding: 0 2px;
}

.tag-input-field {
  flex: 1;
  min-width: 120px;
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  padding: 4px 4px;
}

.tag-input-field:focus { outline: none; }

/* 按钮 */
.form-actions {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 32px;
}

.btn-primary {
  padding: 10px 32px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.error-text { color: var(--tag-rejected-text); font-size: 13px; }
.success-text { color: var(--tag-published-text); font-size: 13px; }
</style>
