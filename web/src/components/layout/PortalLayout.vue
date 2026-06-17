<template>
  <n-layout class="portal-layout">
    <n-layout-header bordered class="portal-header">
      <div class="header-inner">
        <router-link to="/portal/chat" class="logo">OpsMind</router-link>
        <nav class="main-nav">
          <router-link to="/portal/chat" class="nav-link" active-class="nav-link--active">
            智能问答
          </router-link>
          <router-link to="/portal/tickets/submit" class="nav-link" active-class="nav-link--active">
            提交申告
          </router-link>
          <router-link to="/portal/tickets" class="nav-link" active-class="nav-link--active">
            我的申告
          </router-link>
          <router-link to="/portal/messages" class="nav-link nav-link--badge" active-class="nav-link--active">
            消息
            <n-badge
              v-if="appStore.unreadMessageCount > 0"
              :value="appStore.unreadMessageCount"
              :max="99"
              size="tiny"
              class="msg-badge"
            />
          </router-link>
        </nav>
        <div class="header-right">
          <n-button quaternary circle size="small" @click="toggleTheme" title="切换主题">
            <template #icon>
              <n-icon size="18"><SunnyOutline v-if="isDark" /><MoonOutline v-else /></n-icon>
            </template>
          </n-button>
          <n-button text size="small" @click="router.push('/change-password')">修改密码</n-button>
          <n-button text size="small" @click="handleLogout">退出</n-button>
        </div>
      </div>
    </n-layout-header>
    <n-layout-content class="portal-main">
      <router-view />
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { NLayout, NLayoutHeader, NLayoutContent, NButton, NIcon, NBadge } from 'naive-ui'
import { SunnyOutline, MoonOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useTheme } from '@/composables/useTheme'
import { logout as logoutApi } from '@/api/auth'

const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const { toggleTheme, isDark } = useTheme()

let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await appStore.fetchUnreadCount()
  // 每 30 秒轮询未读消息数
  pollTimer = setInterval(() => appStore.fetchUnreadCount(), 30000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

function handleLogout() {
  logoutApi().catch(() => {}) // best-effort
  authStore.clearAuth()
  router.push('/login')
}
</script>

<style scoped>
.portal-layout {
  min-height: 100vh;
}

.portal-header {
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--spacing-lg);
  height: 56px;
  display: flex;
  align-items: center;
  gap: var(--spacing-xl);
}

.logo {
  font-size: 18px;
  font-weight: var(--font-weight-strong);
  color: var(--accent);
  text-decoration: none;
  flex-shrink: 0;
  letter-spacing: -0.3px;
}

.main-nav {
  display: flex;
  gap: var(--spacing-xs);
  flex: 1;
}

.nav-link {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 14px;
  font-weight: var(--font-weight-emphasis);
  padding: 8px 16px;
  border-radius: var(--radius-md);
  transition: color var(--transition-fast), background var(--transition-fast);
  position: relative;
}

.nav-link:hover {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.nav-link--active {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.nav-link--badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.header-right {
  display: flex;
  gap: var(--spacing-sm);
  align-items: center;
  flex-shrink: 0;
}

.portal-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: var(--spacing-xl) var(--spacing-lg);
}
</style>
