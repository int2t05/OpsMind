/**
 * Axios 拦截器增强测试 — P3-19
 *
 * 验证 403 toast 通知和全局 loading 计数器逻辑。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// =============================================================================
// 全局 loading 计数器
// =============================================================================

function createLoadingCounter() {
  let count = 0
  const listeners = new Set<(loading: boolean) => void>()

  function increment() {
    count++
    if (count === 1) notify(true)
  }

  function decrement() {
    if (count > 0) count--
    if (count === 0) notify(false)
  }

  function isLoading(): boolean {
    return count > 0
  }

  function onChange(fn: (loading: boolean) => void) {
    listeners.add(fn)
    return () => listeners.delete(fn)
  }

  function notify(loading: boolean) {
    listeners.forEach((fn) => fn(loading))
  }

  function reset() {
    count = 0
    notify(false)
  }

  return { increment, decrement, isLoading, onChange, reset }
}

// =============================================================================
// 403 处理策略
// =============================================================================

function handle403Response(currentPath: string): {
  showToast: boolean
  message: string
  redirect?: string
} {
  // 无论如何都通知用户
  const base = {
    showToast: true,
    message: '无权限访问该资源',
  }

  // 如果已经在登录页，不再重定向
  if (currentPath === '/login') {
    return base
  }

  return { ...base, redirect: '/login' }
}

describe('全局 loading 计数器 (P3-19)', () => {
  let counter: ReturnType<typeof createLoadingCounter>

  beforeEach(() => {
    counter = createLoadingCounter()
  })

  it('初始状态 isLoading 为 false', () => {
    expect(counter.isLoading()).toBe(false)
  })

  it('increment 后 isLoading 为 true', () => {
    counter.increment()
    expect(counter.isLoading()).toBe(true)
  })

  it('decrement 后 isLoading 恢复 false', () => {
    counter.increment()
    counter.decrement()
    expect(counter.isLoading()).toBe(false)
  })

  it('多次 increment 需要多次 decrement', () => {
    counter.increment()
    counter.increment()
    counter.decrement()
    expect(counter.isLoading()).toBe(true)
    counter.decrement()
    expect(counter.isLoading()).toBe(false)
  })

  it('decrement 不会小于 0', () => {
    counter.decrement()
    counter.decrement()
    expect(counter.isLoading()).toBe(false)
  })

  it('onChange 在状态变化时触发通知', () => {
    const fn = vi.fn()
    counter.onChange(fn)
    counter.increment()
    expect(fn).toHaveBeenCalledWith(true)
    counter.decrement()
    expect(fn).toHaveBeenCalledWith(false)
  })

  it('reset 重置计数并通知', () => {
    counter.increment()
    counter.increment()
    const fn = vi.fn()
    counter.onChange(fn)
    counter.reset()
    expect(counter.isLoading()).toBe(false)
    expect(fn).toHaveBeenCalledWith(false)
  })
})

describe('403 响应处理 (P3-19)', () => {
  it('非登录页收到 403 时应提示并重定向到登录页', () => {
    const result = handle403Response('/admin/dashboard')
    expect(result.showToast).toBe(true)
    expect(result.message).toBe('无权限访问该资源')
    expect(result.redirect).toBe('/login')
  })

  it('已在登录页收到 403 时不应重定向', () => {
    const result = handle403Response('/login')
    expect(result.showToast).toBe(true)
    expect(result.redirect).toBeUndefined()
  })
})
