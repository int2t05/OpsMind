<template>
  <n-layout has-sider class="admin-layout">
    <!-- 侧边栏 -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      :collapsed="appStore.sidebarCollapsed"
      :native-scrollbar="false"
    >
      <div class="sider-header">
        <span v-if="!appStore.sidebarCollapsed" class="logo-text">OpsMind</span>
        <span v-else class="logo-icon">OM</span>
      </div>
      <n-menu
        :collapsed="appStore.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="activeMenu"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>

    <!-- 全局加载进度条 -->
    <div v-if="isLoading" class="global-loading-bar"></div>

    <!-- 主内容区 -->
    <n-layout>
      <n-layout-header bordered class="topbar">
        <div class="topbar-left">
          <n-button quaternary circle size="medium" @click="appStore.toggleSidebar">
            <template #icon>
              <n-icon size="20"><MenuOutline /></n-icon>
            </template>
          </n-button>
        </div>
        <div class="topbar-right">
          <n-button quaternary circle size="medium" @click="toggleTheme">
            <template #icon>
              <n-icon size="20"><SunnyOutline v-if="isDark" /><MoonOutline v-else /></n-icon>
            </template>
          </n-button>
          <n-badge :value="appStore.unreadMessageCount" :max="99" :show="appStore.unreadMessageCount > 0">
            <n-button quaternary circle size="medium" @click="router.push('/portal/messages')">
              <template #icon><n-icon size="20"><NotificationsOutline /></n-icon></template>
            </n-button>
          </n-badge>
          <n-dropdown :options="userDropdownOptions" @select="handleUserDropdown">
            <n-button quaternary>
              <template #icon><n-icon size="18"><PersonOutline /></n-icon></template>
              {{ authStore.user?.real_name || authStore.user?.username }}
            </n-button>
          </n-dropdown>
        </div>
      </n-layout-header>
      <n-layout-content class="content">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent,
  NMenu, NButton, NIcon, NBadge, NDropdown,
} from 'naive-ui'
import {
  GridOutline, TicketOutline, BookOutline,
  PeopleOutline, KeyOutline, SettingsOutline,
  MenuOutline, SunnyOutline, MoonOutline,
  PersonOutline, NotificationsOutline,
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useTheme } from '@/composables/useTheme'
import { useLoading } from '@/composables/useLoading'
import { logout as logoutApi } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()
const { toggleTheme, isDark } = useTheme()
const { isLoading } = useLoading()

// 菜单渲染函数 —— 用 Naive UI Icon 组件渲染 vicons
function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

// 菜单选项
const menuOptions = computed(() => {
  // TODO(layout/AdminLayout): 菜单仍是前端硬编码，登录接口已经返回 menus。
  // 应根据 authStore.menus 渲染，才能真正支持后台动态菜单和按钮权限。
  const items: any[] = [
    { label: '数据看板', key: '/admin/dashboard', icon: renderIcon(GridOutline) },
    { label: '申告管理', key: '/admin/tickets', icon: renderIcon(TicketOutline) },
    { label: '知识库', key: '/admin/knowledge', icon: renderIcon(BookOutline) },
    { label: '用户管理', key: '/admin/users', icon: renderIcon(PeopleOutline) },
    { label: '角色管理', key: '/admin/roles', icon: renderIcon(KeyOutline) },
  ]
  // 仅系统管理员可见系统配置入口
  if (authStore.hasPermission('system:config')) {
    items.push({ label: '系统配置', key: '/admin/config', icon: renderIcon(SettingsOutline) })
  }
  return items
})

// 当前激活菜单项
const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/admin/tickets')) return '/admin/tickets'
  if (path.startsWith('/admin/knowledge')) return '/admin/knowledge'
  if (path.startsWith('/admin/users')) return '/admin/users'
  if (path.startsWith('/admin/roles')) return '/admin/roles'
  if (path.startsWith('/admin/config')
    || path.startsWith('/admin/model-config')
    || path.startsWith('/admin/llm-config')
    || path.startsWith('/admin/audit-logs')) return '/admin/config'
  return '/admin/dashboard'
})

function handleMenuSelect(key: string) {
  router.push(key)
}

// 用户下拉菜单
const userDropdownOptions = [
  { label: '修改密码', key: 'password' },
  { label: '退出登录', key: 'logout' },
]

function handleUserDropdown(key: string) {
  if (key === 'logout') {
    logoutApi().catch(() => {}) // best-effort，不阻塞退出
    authStore.clearAuth()
    router.push('/login')
  } else if (key === 'password') {
    router.push('/change-password')
  }
}
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
}

/* 全局加载进度条 — 基于 useLoading composable */
.global-loading-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--accent);
  z-index: 9999;
  animation: loadingBar 1.5s ease-in-out infinite;
}

@keyframes loadingBar {
  0% { transform: translateX(-100%); }
  50% { transform: translateX(0%); }
  100% { transform: translateX(100%); }
}

.sider-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid var(--border-subtle);
}

.logo-text {
  font-size: 18px;
  font-weight: var(--font-weight-strong);
  color: var(--accent);
  letter-spacing: -0.3px;
}

.logo-icon {
  font-size: 16px;
  font-weight: var(--font-weight-strong);
  color: var(--accent);
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--spacing-md);
  height: 56px;
}

.topbar-left {
  display: flex;
  align-items: center;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.content {
  padding: var(--spacing-lg);
  min-height: calc(100vh - 56px);
}
</style>
