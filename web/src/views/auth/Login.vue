<template>
  <div class="login-page">
    <n-card class="login-card" size="large">
      <template #header>
        <h1 class="login-title">OpsMind</h1>
        <p class="login-subtitle">运维数字员工系统</p>
      </template>
      <n-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
        <n-form-item path="username">
          <n-input
            v-model:value="form.username"
            placeholder="用户名"
            size="large"
            :input-props="{ autocomplete: 'username' }"
          >
            <template #prefix><n-icon :component="PersonOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            placeholder="密码"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'current-password' }"
          >
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item v-if="error" class="error-item">
          <n-alert type="error" :title="error" closable @close="error = ''" />
        </n-form-item>
        <n-form-item>
          <n-button
            type="primary"
            block
            size="large"
            :loading="loading"
            :disabled="loading"
            @click="handleLogin"
          >
            登录
          </n-button>
        </n-form-item>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard, NForm, NFormItem, NInput, NButton, NIcon, NAlert,
} from 'naive-ui'
import { PersonOutline, LockClosedOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { login } from '@/api/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const handleLogin = async () => {
  loading.value = true
  error.value = ''

  try {
    const res = await login(form.value)
    const data = res.data

    authStore.setToken(data.access_token, data.refresh_token)
    authStore.setUserInfo({
      user: data.user,
      roles: data.roles,
      permissions: data.permissions,
      menus: data.menus,
    })

    if (data.permissions?.length > 0) {
      router.push('/admin')
    } else {
      router.push('/portal')
    }
  } catch (err: any) {
    error.value = err?.message || '登录失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--spacing-lg);
}

.login-card {
  width: 100%;
  max-width: 400px;
}

.login-title {
  font-size: 28px;
  font-weight: var(--font-weight-strong);
  color: var(--accent);
  text-align: center;
  letter-spacing: -0.5px;
  margin: 0;
}

.login-subtitle {
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
</style>
