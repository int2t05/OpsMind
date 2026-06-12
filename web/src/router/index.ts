/**
 * Vue Router 路由定义
 *
 * 路由分为三组：
 * - 公开路由（/login）— 无需认证
 * - 门户路由（/portal/*）— 需要登录 + 报障人角色
 * - 后台路由（/admin/*）— 需要登录 + 对应角色权限
 *
 * 路由守卫检查 token 有效性、首次登录强制跳转修改密码页、角色权限校验。
 *
 * TODO(router): 路由守卫未读取 meta.roles 进行角色权限校验 — 任何已登录用户可访问所有路由。
 *               需在 beforeEach 中根据 auth store 的 user roles 与 to.meta.roles 做交集判断。
 * TODO(router): 路由守卫仅检查 token 是否存在，未校验 JWT 是否过期 — 过期 token 会导致页面加载闪白后
 *               才被 API 拦截器踢回登录页。应在守卫层主动解码 token 并校验 exp。
 * TODO(router): 缺少 scrollBehavior 配置 — 页面间导航不会自动恢复/重置滚动位置。
 * TODO(router): /admin 路由组缺少 meta.roles 定义 — 应限定为 admin/operator 等后台角色。
 */

import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '@/utils/auth'

const router = createRouter({
  history: createWebHistory(),
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
      meta: { requiresAuth: true },
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
  }

  next()
})

export default router
