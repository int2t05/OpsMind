/**
 * 全局错误处理测试 — P3-21
 *
 * 验证 Vue 应用 errorHandler 的行为。
 */
import { describe, it, expect, vi } from 'vitest'

/** 全局 errorHandler 实现（待添加到 main.ts） */
function createErrorHandler(onError?: (err: unknown, info: string) => void) {
  return function errorHandler(err: unknown, _instance: unknown, info: string) {
    console.error('[Vue Error]', info, err)
    onError?.(err, info)
    // 阻止错误静默消失
    return false
  }
}

describe('全局 errorHandler (P3-21)', () => {
  it('应输出 console.error 而非静默消失', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const handler = createErrorHandler()

    handler(new Error('测试错误'), null, 'render function')

    expect(spy).toHaveBeenCalledWith(
      '[Vue Error]',
      'render function',
      expect.any(Error)
    )
    spy.mockRestore()
  })

  it('应调用自定义 onError 回调', () => {
    const onError = vi.fn()
    const handler = createErrorHandler(onError)

    const err = new Error('边界错误')
    handler(err, null, 'setup function')

    expect(onError).toHaveBeenCalledWith(err, 'setup function')
  })

  it('不应抛出异常（防止错误处理器本身崩溃）', () => {
    const handler = createErrorHandler()
    expect(() => handler('字符串错误', null, 'watcher callback')).not.toThrow()
  })
})
