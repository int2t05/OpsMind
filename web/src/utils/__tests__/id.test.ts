/**
 * 唯一 ID 生成工具测试 — P1-10
 *
 * 为 Chat.vue 消息列表的 v-for key 提供唯一标识，
 * 替代不可靠的数组索引 :key="i"。
 */
import { describe, it, expect } from 'vitest'

/** 生成简短的唯一 ID（待提取到 utils/id.ts） */
function generateId(): string {
  // 使用 crypto.randomUUID() 在当前环境的回退方案
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // jsdom 回退
  return `id_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`
}

describe('generateId', () => {
  it('返回非空字符串', () => {
    const id = generateId()
    expect(id).toBeTruthy()
    expect(typeof id).toBe('string')
  })

  it('连续调用生成不同 ID', () => {
    const ids = new Set<string>()
    for (let i = 0; i < 100; i++) {
      ids.add(generateId())
    }
    expect(ids.size).toBe(100)
  })

  it('ID 长度至少 8 字符', () => {
    for (let i = 0; i < 20; i++) {
      expect(generateId().length).toBeGreaterThanOrEqual(8)
    }
  })
})
