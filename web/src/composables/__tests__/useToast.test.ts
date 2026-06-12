/**
 * useToast composable 测试 — P2-12
 *
 * 测试共享 toast 状态管理逻辑。
 * 运行前：import 将失败（文件不存在）。
 * 实现后：验证 show/hide、自动消失、定时器清理。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { nextTick } from 'vue'
import { useToast } from '@/composables/useToast'

describe('useToast composable', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('初始状态 visible 为 false', () => {
    const toast = useToast()
    expect(toast.visible.value).toBe(false)
    expect(toast.message.value).toBe('')
  })

  it('showToast 设置 visible 为 true 并设置消息', async () => {
    const toast = useToast()
    toast.showToast('操作成功', 'success')
    await nextTick()
    expect(toast.visible.value).toBe(true)
    expect(toast.message.value).toBe('操作成功')
    expect(toast.type.value).toBe('success')
  })

  it('默认 duration 后自动隐藏', async () => {
    const toast = useToast(2000)
    toast.showToast('保存成功')
    await nextTick()
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(2000)
    await nextTick()
    expect(toast.visible.value).toBe(false)
  })

  it('连续调用 showToast 重置定时器', async () => {
    const toast = useToast(2000)
    toast.showToast('第一条')
    await nextTick()
    vi.advanceTimersByTime(1500)
    toast.showToast('第二条')
    await nextTick()
    vi.advanceTimersByTime(1500)
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(500)
    await nextTick()
    expect(toast.visible.value).toBe(false)
  })

  it('clearToast 清理定时器', async () => {
    const toast = useToast()
    toast.showToast('消息')
    await nextTick()
    expect(() => toast.clearToast()).not.toThrow()
  })

  it('clearToast 多次调用不抛出', () => {
    const toast = useToast()
    expect(() => {
      toast.clearToast()
      toast.clearToast()
    }).not.toThrow()
  })
})
