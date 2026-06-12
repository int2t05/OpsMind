/**
 * 共享工具函数测试 — P2-11
 *
 * 这些函数将从各视图提取到独立的工具模块中。
 * 运行前：所有 import 将失败（文件不存在）。
 * 实现后：测试验证所有提取的函数行为一致。
 */
import { describe, it, expect } from 'vitest'
import { urgencyText, ticketStatusClass } from '@/utils/ticket'
import { knowledgeStatusText, knowledgeStatusClass, processText, processClass } from '@/utils/knowledge'
import { formatDate } from '@/utils/format'

describe('urgencyText (utils/ticket.ts)', () => {
  it('urgency=1 返回"低"', () => { expect(urgencyText(1)).toBe('低') })
  it('urgency=2 返回"中"', () => { expect(urgencyText(2)).toBe('中') })
  it('urgency=3 返回"高"', () => { expect(urgencyText(3)).toBe('高') })
  it('未知值返回"未知"', () => { expect(urgencyText(99)).toBe('未知') })
  it('零值返回"未知"', () => { expect(urgencyText(0)).toBe('未知') })
})

describe('ticketStatusClass (utils/ticket.ts)', () => {
  it('status=0 返回 status-pending', () => { expect(ticketStatusClass(0)).toBe('status-pending') })
  it('status=1 返回 status-processing', () => { expect(ticketStatusClass(1)).toBe('status-processing') })
  it('status=3 返回 status-resolved', () => { expect(ticketStatusClass(3)).toBe('status-resolved') })
  it('未知值返回空字符串', () => { expect(ticketStatusClass(99)).toBe('') })
})

describe('knowledgeStatusText (utils/knowledge.ts)', () => {
  it('status=0 返回"草稿"', () => { expect(knowledgeStatusText(0)).toBe('草稿') })
  it('status=1 返回"待审核"', () => { expect(knowledgeStatusText(1)).toBe('待审核') })
  it('status=2 返回"已发布"', () => { expect(knowledgeStatusText(2)).toBe('已发布') })
  it('status=3 返回"已禁用"', () => { expect(knowledgeStatusText(3)).toBe('已禁用') })
})

describe('knowledgeStatusClass (utils/knowledge.ts)', () => {
  it('各状态映射正确', () => {
    expect(knowledgeStatusClass(0)).toBe('status-draft')
    expect(knowledgeStatusClass(2)).toBe('status-published')
    expect(knowledgeStatusClass(3)).toBe('status-disabled')
  })
})

describe('processText (utils/knowledge.ts)', () => {
  it('status=0 返回"待处理"', () => { expect(processText(0)).toBe('待处理') })
  it('status=2 返回"已完成"', () => { expect(processText(2)).toBe('已完成') })
  it('status=3 返回"失败"', () => { expect(processText(3)).toBe('失败') })
})

describe('processClass (utils/knowledge.ts)', () => {
  it('各处理状态映射正确', () => {
    expect(processClass(1)).toBe('process-processing')
    expect(processClass(2)).toBe('process-done')
    expect(processClass(3)).toBe('process-failed')
  })
})

describe('formatDate (utils/format.ts)', () => {
  it('空字符串返回"-"', () => { expect(formatDate('')).toBe('-') })
  it('有效日期返回格式化字符串', () => {
    const result = formatDate('2026-01-15T10:30:00Z')
    expect(result).toContain('2026')
    expect(result).toContain('01')
  })
  it('不抛出异常', () => {
    expect(() => formatDate('invalid-date')).not.toThrow()
  })
})
