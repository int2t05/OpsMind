/**
 * 日期/文本格式化工具
 *
 * 统一项目中所有视图的日期格式化逻辑，消除几乎所有视图中重复的 formatDate 定义。
 */

/** 统一日期格式化（中文 locale） */
export function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
