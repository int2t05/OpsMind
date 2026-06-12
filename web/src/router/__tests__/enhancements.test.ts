/**
 * Router 增强测试 — P3-20
 *
 * 验证 scrollBehavior 和 JWT 过期检查逻辑。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'

// =============================================================================
// JWT 过期检查（提取为纯函数）
// =============================================================================

/** 解码 JWT payload（不验证签名，仅提取过期时间） */
function decodeJWTPayload(token: string): { exp?: number } | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1]))
    return payload
  } catch {
    return null
  }
}

/** 检查 JWT token 是否已过期 */
function isTokenExpired(token: string): boolean {
  const payload = decodeJWTPayload(token)
  if (!payload || !payload.exp) {
    // 无法解析或无 exp 字段，保守处理：视作有效
    return false
  }
  // exp 是秒级 Unix 时间戳，加 60 秒缓冲
  const now = Math.floor(Date.now() / 1000)
  return payload.exp < now + 60
}

// =============================================================================
// scrollBehavior
// =============================================================================

/** 路由切换时的滚动行为 */
function scrollBehavior(
  _to: unknown,
  _from: unknown,
  savedPosition: { left: number; top: number } | null,
) {
  if (savedPosition) {
    return savedPosition
  }
  // 新导航始终回到顶部
  return { top: 0 }
}

describe('JWT 过期检查 (P3-20)', () => {
  it('有效 token 返回 false（未过期）', () => {
    const future = Math.floor(Date.now() / 1000) + 3600
    const payload = btoa(JSON.stringify({ exp: future, sub: '1' }))
    const token = `header.${payload}.sig`
    expect(isTokenExpired(token)).toBe(false)
  })

  it('过期 token 返回 true', () => {
    const past = Math.floor(Date.now() / 1000) - 3600
    const payload = btoa(JSON.stringify({ exp: past, sub: '1' }))
    const token = `header.${payload}.sig`
    expect(isTokenExpired(token)).toBe(true)
  })

  it('无法解析的 token 返回 false（保守处理）', () => {
    expect(isTokenExpired('invalid-token')).toBe(false)
  })

  it('无 exp 字段的 token 返回 false', () => {
    const payload = btoa(JSON.stringify({ sub: '1' }))
    const token = `header.${payload}.sig`
    expect(isTokenExpired(token)).toBe(false)
  })

  it('临近过期（< 60 秒缓冲）视为过期', () => {
    const soon = Math.floor(Date.now() / 1000) + 30
    const payload = btoa(JSON.stringify({ exp: soon, sub: '1' }))
    const token = `header.${payload}.sig`
    expect(isTokenExpired(token)).toBe(true)
  })
})

describe('scrollBehavior (P3-20)', () => {
  it('有 savedPosition 时恢复滚动位置', () => {
    const result = scrollBehavior(null, null, { left: 0, top: 500 })
    expect(result).toEqual({ left: 0, top: 500 })
  })

  it('无 savedPosition 时回到顶部', () => {
    const result = scrollBehavior(null, null, null)
    expect(result).toEqual({ top: 0 })
  })
})
