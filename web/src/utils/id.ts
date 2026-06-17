/** 生成简短的唯一 ID，带有 crypto.randomUUID() 回退 */
export function generateId(): string {
  // 使用 crypto.randomUUID() 在当前环境的回退方案
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // HTTP/localhost 等环境回退
  return `id_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`
}
