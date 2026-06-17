/**
 * 共享工具函数测试
 *
 * 验证从各视图提取到 utils/ 模块的函数行为一致。
 */
import { describe, it, expect } from 'vitest'
import { urgencyText, urgencyClass, ticketStatusClass, scopeText, actionText } from '@/utils/ticket'
import { articleStatusText, articleStatusClass, processText, processClass } from '@/utils/knowledge'
import { formatDate } from '@/utils/date'

// ═══════════════════════════════════════════════════════════════
// ticket.ts
// ═══════════════════════════════════════════════════════════════

describe('urgencyText', () => {
  it('1→低, 2→中, 3→高', () => {
    expect(urgencyText(1)).toBe('低')
    expect(urgencyText(2)).toBe('中')
    expect(urgencyText(3)).toBe('高')
  })
  it('未知值返回"未知"', () => { expect(urgencyText(99)).toBe('未知') })
})

describe('urgencyClass', () => {
  it('3→high, 2→medium, 其他→low', () => {
    expect(urgencyClass(3)).toBe('high')
    expect(urgencyClass(2)).toBe('medium')
    expect(urgencyClass(1)).toBe('low')
  })
})

describe('ticketStatusClass', () => {
  it('状态映射', () => {
    expect(ticketStatusClass(1)).toBe('pending')
    expect(ticketStatusClass(2)).toBe('processing')
    expect(ticketStatusClass(3)).toBe('supplement')
    expect(ticketStatusClass(4)).toBe('resolved')
    expect(ticketStatusClass(5)).toBe('closed')
  })
})

describe('scopeText', () => {
  it('1→个人, 2→部门, 3→全公司', () => {
    expect(scopeText(1)).toBe('个人')
    expect(scopeText(2)).toBe('部门')
    expect(scopeText(3)).toBe('全公司')
  })
})

describe('actionText', () => {
  it('已定义操作映射', () => {
    expect(actionText('start')).toBe('开始处理')
    expect(actionText('resolve')).toBe('已解决')
    expect(actionText('close')).toBe('关闭')
  })
  it('未知操作返回原值', () => { expect(actionText('unknown_action')).toBe('unknown_action') })
})

// ═══════════════════════════════════════════════════════════════
// knowledge.ts
// ═══════════════════════════════════════════════════════════════

describe('articleStatusText', () => {
  it('状态映射 (1-6)', () => {
    expect(articleStatusText(1)).toBe('草稿')
    expect(articleStatusText(2)).toBe('待审核')
    expect(articleStatusText(3)).toBe('已通过')
    expect(articleStatusText(4)).toBe('已发布')
    expect(articleStatusText(5)).toBe('已停用')
    expect(articleStatusText(6)).toBe('已驳回')
  })
})

describe('articleStatusClass', () => {
  it('CSS 类映射', () => {
    expect(articleStatusClass(1)).toBe('draft')
    expect(articleStatusClass(2)).toBe('pending')
    expect(articleStatusClass(4)).toBe('published')
    expect(articleStatusClass(6)).toBe('rejected')
  })
})

describe('processText', () => {
  it('undefined→"-"', () => { expect(processText(undefined)).toBe('-') })
  it('pending→待处理, completed→已完成, failed→失败', () => {
    expect(processText('pending')).toBe('待处理')
    expect(processText('completed')).toBe('已完成')
    expect(processText('failed')).toBe('失败')
  })
})

describe('processClass', () => {
  it('undefined→""', () => { expect(processClass(undefined)).toBe('') })
  it('parsing→pending, completed→completed, failed→failed', () => {
    expect(processClass('parsing')).toBe('pending')
    expect(processClass('completed')).toBe('completed')
    expect(processClass('failed')).toBe('failed')
  })
})

// ═══════════════════════════════════════════════════════════════
// date.ts
// ═══════════════════════════════════════════════════════════════

describe('formatDate', () => {
  it('空字符串返回"-"', () => { expect(formatDate('')).toBe('-') })
  it('有效日期返回格式化字符串', () => {
    const result = formatDate('2026-01-15T10:30:00Z')
    expect(result).toContain('2026')
    expect(result).toContain('01')
  })
  it('无效日期不抛出异常', () => {
    expect(() => formatDate('invalid-date')).not.toThrow()
  })
})
