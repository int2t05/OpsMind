/**
 * Vue Router 路由定义
 *
 * 路由分为三组：
 * - 公开路由（/login）— 无需认证
 * - 门户路由（/portal/*）— 需要登录 + 报障人角色
 * - 后台路由（/admin/*）— 需要登录 + 对应角色权限
 *
 * 路由守卫检查 token 有效性、JWT 过期、角色权限校验。
 */

import { createRouter, createWebHistory } from 'vue-router'
import { getToken, removeToken } from '@/utils/auth'
import { useAuthStore } from '@/stores/auth'

/** 解码 JWT payload（不验证签名，仅提取过期时间） */
function decodeJWTPayload(token: string): { exp?: number } | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    return JSON.parse(atob(parts[1]))
  } catch {
    return null
  }
}

/** 检查 JWT token 是否已过期（含 60 秒缓冲） */
function isTokenExpired(token: string): boolean {
  const payload = decodeJWTPayload(token)
  if (!payload?.exp) return false
  const now = Math.floor(Date.now() / 1000)
  return payload.exp < now + 60
}

const router = createRouter({
  history: createWebHistory(),
  // 页面切换时恢复保存的滚动位置，否则回到顶部
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    return { top: 0 }
  },
  routes: [
    // 公开路由
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/auth/Login.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/change-password',
      name: 'ChangePassword',
      component: () => import('@/views/auth/ChangePassword.vue'),
      meta: { requiresAuth: true }
    },

    // 门户路由
    {
      path: '/portal',
      component: () => import('@/components/layout/PortalLayout.vue'),
      meta: { requiresAuth: true, roles: ['reporter'] },
      children: [
        {
          path: '',
          redirect: '/portal/chat'
        },
        {
          path: 'chat',
          name: 'PortalChat',
          component: () => import('@/views/portal/Chat.vue')
        },
        {
          path: 'tickets',
          name: 'PortalTickets',
          component: () => import('@/views/portal/TicketQuery.vue')
        },
        {
          path: 'tickets/submit',
          name: 'PortalTicketSubmit',
          component: () => import('@/views/portal/TicketSubmit.vue')
        },
        {
          path: 'tickets/:id',
          name: 'PortalTicketDetail',
          component: () => import('@/views/portal/TicketDetail.vue')
        },
        {
          path: 'messages',
          name: 'PortalMessages',
          component: () => import('@/views/portal/Messages.vue')
        }
      ]
    },

    // 后台路由
    {
      path: '/admin',
      component: () => import('@/components/layout/AdminLayout.vue'),
      meta: { requiresAuth: true, roles: ['admin', 'operator', 'knowledge_manager'] },
      children: [
        {
          path: '',
          redirect: '/admin/dashboard'
        },
        {
          path: 'dashboard',
          name: 'AdminDashboard',
          component: () => import('@/views/admin/Dashboard.vue')
        },
        {
          path: 'tickets',
          name: 'AdminTickets',
          component: () => import('@/views/admin/TicketList.vue')
        },
        {
          path: 'tickets/:id',
          name: 'AdminTicketDetail',
          component: () => import('@/views/admin/TicketDetail.vue')
        },
        {
          path: 'knowledge',
          name: 'AdminKnowledge',
          component: () => import('@/views/admin/KnowledgeList.vue')
        },
        {
          path: 'knowledge/:id',
          name: 'AdminKnowledgeEdit',
          component: () => import('@/views/admin/KnowledgeEdit.vue')
        },
        {
          path: 'users',
          name: 'AdminUsers',
          component: () => import('@/views/admin/UserList.vue')
        },
        {
          path: 'roles',
          name: 'AdminRoles',
          component: () => import('@/views/admin/RoleManage.vue')
        },
        {
          path: 'audit-logs',
          name: 'AdminAuditLogs',
          component: () => import('@/views/admin/AuditLog.vue')
        },
        {
          path: 'model-config',
          name: 'AdminModelConfig',
          component: () => import('@/views/admin/ModelConfig.vue')
        },
        {
          path: 'llm-config',
          name: 'AdminLLMConfig',
          component: () => import('@/views/admin/LLMConfig.vue')
        },
        {
          path: 'embedding-config',
          redirect: '/admin/llm-config'
        },
        {
          path: 'config',
          name: 'AdminSystemConfig',
          component: () => import('@/views/admin/SystemConfig.vue')
        }
      ]
    },

    // 默认重定向
    {
      path: '/',
      redirect: '/login'
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/login'
    }
  ]
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const token = getToken()

  // 需要认证的路由
  if (to.meta.requiresAuth !== false) {
    if (!token) {
      // 未登录，跳转登录页
      next('/login')
      return
    }

    // JWT 过期检查：过期 token 在守卫层主动拦截，避免页面闪白
    if (isTokenExpired(token)) {
      removeToken()
      next('/login')
      return
    }

    // 角色权限校验：若路由定义了 meta.roles，检查用户是否拥有所需角色
    const requiredRoles = to.meta.roles as string[] | undefined
    if (requiredRoles && requiredRoles.length > 0) {
      const authStore = useAuthStore()
      const hasRole = requiredRoles.some((role) => authStore.roles.includes(role))
      if (!hasRole) {
        // 角色不匹配，重定向到登录页（与缺少 token 行为一致）
        next('/login')
        return
      }
    }
  }

  next()
})

export default router
