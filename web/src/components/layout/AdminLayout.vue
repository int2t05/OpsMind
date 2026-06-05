<template>
  <div class="admin-layout" :class="{ collapsed: appStore.sidebarCollapsed }">
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2 class="logo">OpsMind</h2>
      </div>
      <nav class="sidebar-nav">
        <router-link to="/admin" class="nav-item">
          <span class="nav-icon">📊</span>
          <span class="nav-text">看板</span>
        </router-link>
        <router-link to="/admin/tickets" class="nav-item">
          <span class="nav-icon">🎫</span>
          <span class="nav-text">申告管理</span>
        </router-link>
        <router-link to="/admin/knowledge" class="nav-item">
          <span class="nav-icon">📚</span>
          <span class="nav-text">知识库</span>
        </router-link>
        <router-link to="/admin/users" class="nav-item">
          <span class="nav-icon">👥</span>
          <span class="nav-text">用户管理</span>
        </router-link>
        <router-link to="/admin/roles" class="nav-item">
          <span class="nav-icon">🔑</span>
          <span class="nav-text">角色管理</span>
        </router-link>
        <router-link to="/admin/config" class="nav-item">
          <span class="nav-icon">⚙️</span>
          <span class="nav-text">系统配置</span>
        </router-link>
      </nav>
    </aside>
    <main class="main-content">
      <header class="topbar">
        <button class="btn-toggle" @click="appStore.toggleSidebar">☰</button>
        <div class="topbar-right">
          <span class="username">{{ authStore.user?.real_name || authStore.user?.username }}</span>
          <span class="badge" v-if="appStore.unreadMessageCount">{{ appStore.unreadMessageCount }}</span>
          <button class="btn-logout" @click="handleLogout">退出</button>
        </div>
      </header>
      <div class="content">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useAppStore } from '../../stores/app'

const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const handleLogout = () => {
  authStore.clearAuth()
  router.push('/login')
}
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg-base);
}

.sidebar {
  width: 240px;
  background: var(--bg-elevated);
  border-right: 1px solid var(--border);
  transition: width 0.2s;
}

.collapsed .sidebar {
  width: 64px;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.logo {
  font-size: 18px;
  font-weight: 700;
  color: var(--accent);
  margin: 0;
}

.collapsed .logo {
  font-size: 14px;
  text-align: center;
}

.sidebar-nav {
  padding: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 14px;
}

.nav-item:hover {
  background: var(--bg-subtle);
  color: var(--text-primary);
}

.nav-item.router-link-active {
  background: var(--accent);
  color: white;
}

.nav-icon {
  font-size: 16px;
}

.collapsed .nav-text {
  display: none;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border);
}

.btn-toggle {
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 20px;
  cursor: pointer;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.username {
  font-size: 14px;
  color: var(--text-secondary);
}

.badge {
  background: #f87171;
  color: white;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 12px;
}

.btn-logout {
  padding: 6px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text-primary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.content {
  flex: 1;
  padding: 24px;
}
</style>
