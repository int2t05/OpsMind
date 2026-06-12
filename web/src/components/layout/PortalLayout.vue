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
              v-if="unreadCount > 0"
              :value="unreadCount"
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
// TODO(layout/PortalLayout): catch 块静默失败 — getUnreadCount 调用失败时用户界面无任何提示。
// TODO(layout/PortalLayout): 使用 (res as any) 强制类型转换 — 待 API 层泛型补全后移除。
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NLayout, NLayoutHeader, NLayoutContent, NButton, NIcon, NBadge } from 'naive-ui'
import { SunnyOutline, MoonOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import { getUnreadCount } from '@/api/message'

const router = useRouter()
const authStore = useAuthStore()
const { toggleTheme, isDark } = useTheme()
const unreadCount = ref(0)

onMounted(async () => {
  try {
    const res = await getUnreadCount()
    const data = (res as any).data || res
    unreadCount.value = data?.count ?? data ?? 0
  } catch {
    // 静默失败
  }
})

function handleLogout() {
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
