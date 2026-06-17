/**
 * 路由守卫测试 — P0-2
 *
 * 验证路由守卫的权限校验逻辑：
 * 1. 提取角色匹配检查为可测试的纯函数
 * 2. 验证 token 缺失时跳转登录页
 * 3. 验证角色不匹配时拒绝访问
 */
import { describe, it, expect } from 'vitest'

// 被测试的守卫逻辑（提取为纯函数，便于单元测试）
function checkRouteAccess(params: {
  hasToken: boolean
  requiresAuth: boolean | undefined
  requiredRoles: string[] | undefined
  userRoles: string[]
}): { allowed: boolean; redirect?: string } {
  const { hasToken, requiresAuth, requiredRoles, userRoles } = params

  // 无需认证的路由 — 直接放行
  if (requiresAuth === false) {
    return { allowed: true }
  }

  // 需要认证但无 token — 跳转登录页
  if (!hasToken) {
    return { allowed: false, redirect: '/login' }
  }

  // 需要特定角色 — 检查用户是否拥有至少一个所需角色
  if (requiredRoles && requiredRoles.length > 0) {
    const hasRole = requiredRoles.some((role) => userRoles.includes(role))
    if (!hasRole) {
      return { allowed: false, redirect: '/login' }
    }
  }

  return { allowed: true }
}

describe('路由守卫权限校验逻辑', () => {
  describe('无需认证的路由 (requiresAuth === false)', () => {
    it('未登录用户可访问登录页', () => {
      const result = checkRouteAccess({
        hasToken: false,
        requiresAuth: false,
        requiredRoles: undefined,
        userRoles: [],
      })
      expect(result.allowed).toBe(true)
    })

    it('已登录用户也可访问登录页', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: false,
        requiredRoles: undefined,
        userRoles: ['admin'],
      })
      expect(result.allowed).toBe(true)
    })
  })

  describe('需要认证的路由', () => {
    it('无 token 时应重定向到 /login', () => {
      const result = checkRouteAccess({
        hasToken: false,
        requiresAuth: true,
        requiredRoles: undefined,
        userRoles: [],
      })
      expect(result.allowed).toBe(false)
      expect(result.redirect).toBe('/login')
    })

    it('有 token 但无角色要求时应放行', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: undefined,
        userRoles: ['reporter'],
      })
      expect(result.allowed).toBe(true)
    })
  })

  describe('meta.roles 角色权限校验', () => {
    it('用户角色匹配时应放行', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: ['reporter'],
        userRoles: ['reporter'],
      })
      expect(result.allowed).toBe(true)
    })

    it('用户角色不匹配时应拒绝', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: ['admin'],
        userRoles: ['reporter'],
      })
      expect(result.allowed).toBe(false)
    })

    it('用户拥有多个角色，其中之一匹配时应放行', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: ['admin', 'operator'],
        userRoles: ['reporter', 'operator'],
      })
      expect(result.allowed).toBe(true)
    })

    it('requiredRoles 为空数组时视为无角色限制', () => {
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: [],
        userRoles: ['reporter'],
      })
      expect(result.allowed).toBe(true)
    })

    it('portal 路由需要 reporter 角色', () => {
      // 模拟 portal 场景：已登录但无 reporter 角色
      const result = checkRouteAccess({
        hasToken: true,
        requiresAuth: true,
        requiredRoles: ['reporter'],
        userRoles: [], // 没有 reporter 角色
      })
      expect(result.allowed).toBe(false)
    })
  })
})
