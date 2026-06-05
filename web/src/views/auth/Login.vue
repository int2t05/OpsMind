<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">OpsMind 运维数字员工</h1>
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username">用户名</label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            placeholder="请输入用户名"
            required
          />
        </div>
        <div class="form-group">
          <label for="password">密码</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            required
          />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <button type="submit" class="btn-login" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { login } from '../../api/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({
  username: '',
  password: '',
})

const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''

  try {
    const res = await login(form.value)
    const data = res.data.data

    authStore.setToken(data.access_token)
    authStore.setUserInfo({
      user: data.user,
      roles: data.roles,
      permissions: data.permissions,
      menus: data.menus,
    })

    // 首次登录跳转修改密码页
    if (data.user.first_login) {
      router.push('/change-password')
    } else {
      router.push('/admin')
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '登录失败'
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
  background: var(--bg-base);
}

.login-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 32px;
  width: 100%;
  max-width: 400px;
}

.login-title {
  text-align: center;
  margin-bottom: 24px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.form-group {
  margin-bottom: 16px;
}

label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: var(--text-secondary);
}

input {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 14px;
}

input:focus {
  outline: none;
  border-color: var(--accent);
}

.error-message {
  margin-bottom: 16px;
  padding: 8px 12px;
  background: #3a1a1a;
  color: #f87171;
  border-radius: 4px;
  font-size: 14px;
}

.btn-login {
  width: 100%;
  padding: 10px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.btn-login:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
