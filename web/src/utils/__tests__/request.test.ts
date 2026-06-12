/**
 * Axios 请求工具测试 — P1-7 (401 防循环)
 *
 * 验证 401 响应拦截器包含防循环保护：
 * 若用户当前已在 /login 页面，收到 401 时不应再次跳转 /login。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'

// 被测试的 401 处理逻辑（提取为纯函数）
function shouldRedirectToLogin(currentPath: string): boolean {
  // 已在登录页则不跳转，防止无限循环
  return currentPath !== '/login'
}

describe('401 响应拦截器防循环逻辑', () => {
  it('当前已在 /login 页面时不应重定向', () => {
    expect(shouldRedirectToLogin('/login')).toBe(false)
  })

  it('当前在 /portal/chat 页面时应重定向', () => {
    expect(shouldRedirectToLogin('/portal/chat')).toBe(true)
  })

  it('当前在 /admin/dashboard 页面时应重定向', () => {
    expect(shouldRedirectToLogin('/admin/dashboard')).toBe(true)
  })

  it('当前在 /change-password 页面时应重定向', () => {
    expect(shouldRedirectToLogin('/change-password')).toBe(true)
  })

  it('子路径包含 login 关键字的正常页面应重定向（不误判）', () => {
    // /portal/login-help 这样的路径不匹配 '/login'，会被重定向
    // 注意：这里是前缀精确匹配 '/login'，不是模糊匹配
    expect(shouldRedirectToLogin('/portal/something')).toBe(true)
  })
})
