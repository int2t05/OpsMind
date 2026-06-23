<template>
  <div class="page">
    <n-card class="card" size="large">
      <template #header>
        <h1 class="title">重置密码</h1>
        <p class="subtitle">输入用户名和新密码即可重置</p>
      </template>
      <n-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleSubmit">
        <n-form-item path="username">
          <n-input
            v-model:value="form.username"
            placeholder="用户名"
            size="large"
          >
            <template #prefix><n-icon :component="PersonOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="new_password">
          <n-input
            v-model:value="form.new_password"
            type="password"
            placeholder="新密码（8-32位，含大小写字母和数字）"
            size="large"
            show-password-on="click"
          >
            <template #prefix><n-icon :component="LockOpenOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="confirmPassword">
          <n-input
            v-model:value="confirmPassword"
            type="password"
            placeholder="确认新密码"
            size="large"
            show-password-on="click"
          >
            <template #prefix><n-icon :component="ShieldCheckmarkOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item v-if="error" class="error-item">
          <n-alert type="error" :title="error" closable @close="error = ''" />
        </n-form-item>
        <n-form-item v-if="success" class="error-item">
          <n-alert type="success" title="密码重置成功，请返回登录" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" block size="large" :loading="loading" @click="handleSubmit">
            重置密码
          </n-button>
        </n-form-item>
        <n-form-item>
          <div class="back-link">
            <n-button text type="primary" @click="router.push('/login')">
              返回登录
            </n-button>
          </div>
        </n-form-item>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NIcon, NAlert } from 'naive-ui'
import { PersonOutline, LockOpenOutline, ShieldCheckmarkOutline } from '@vicons/ionicons5'
import { resetPassword } from '@/api/auth'

const router = useRouter()

const form = ref({ username: '', new_password: '' })
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)

const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$/

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    {
      validator: (_: any, value: string) => passwordRegex.test(value),
      message: '密码需8-32位，包含大小写字母和数字',
      trigger: 'blur',
    },
  ],
}

const handleSubmit = async () => {
  error.value = ''
  success.value = false

  if (form.value.new_password !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true
  try {
    await resetPassword(form.value)
    success.value = true
    form.value = { username: '', new_password: '' }
    confirmPassword.value = ''
  } catch (err: any) {
    error.value = err.response?.data?.message || '重置失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--spacing-lg);
}

.card {
  width: 100%;
  max-width: 400px;
}

.title {
  font-size: 20px;
  font-weight: var(--font-weight-strong);
  text-align: center;
  margin: 0;
}

.subtitle {
  font-size: 14px;
  color: var(--text-tertiary);
  text-align: center;
  margin-top: var(--spacing-xs);
}

:deep(.n-card-header) {
  padding-bottom: 0;
  border-bottom: none;
}

.error-item :deep(.n-form-item-blank) {
  width: 100%;
}

.back-link {
  text-align: center;
}
</style>
