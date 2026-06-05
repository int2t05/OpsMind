<template>
  <div class="change-password-page">
    <div class="card">
      <h1 class="title">修改密码</h1>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="oldPassword">旧密码</label>
          <input
            id="oldPassword"
            v-model="form.old_password"
            type="password"
            placeholder="请输入旧密码"
            required
          />
        </div>
        <div class="form-group">
          <label for="newPassword">新密码</label>
          <input
            id="newPassword"
            v-model="form.new_password"
            type="password"
            placeholder="8-32位，含大小写字母和数字"
            required
          />
        </div>
        <div class="form-group">
          <label for="confirmPassword">确认密码</label>
          <input
            id="confirmPassword"
            v-model="confirmPassword"
            type="password"
            placeholder="请再次输入新密码"
            required
          />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <button type="submit" class="btn-submit" :disabled="loading">
          {{ loading ? '提交中...' : '确认修改' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { changePassword } from '../../api/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({
  old_password: '',
  new_password: '',
})

const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')

const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$/

const handleSubmit = async () => {
  error.value = ''

  if (!passwordRegex.test(form.value.new_password)) {
    error.value = '密码需8-32位，包含大小写字母和数字'
    return
  }

  if (form.value.new_password !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true

  try {
    await changePassword(form.value)
    authStore.clearAuth()
    router.push('/login')
  } catch (err: any) {
    error.value = err.response?.data?.message || '修改失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.change-password-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg-base);
}

.card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 32px;
  width: 100%;
  max-width: 400px;
}

.title {
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

.btn-submit {
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

.btn-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
