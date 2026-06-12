/**
 * 知识库相关共享工具函数
 *
 * 从 admin/KnowledgeList、admin/KnowledgeEdit 两个视图中提取，
 * 消除 statusText/statusClass/processText/processClass 的重复定义。
 */

/** 知识文章状态文本 */
export function knowledgeStatusText(status: number): string {
  const map: Record<number, string> = {
    0: '草稿',
    1: '待审核',
    2: '已发布',
    3: '已禁用',
  }
  return map[status] || '未知'
}

/** 知识文章状态 CSS 类 */
export function knowledgeStatusClass(status: number): string {
  const map: Record<number, string> = {
    0: 'status-draft',
    1: 'status-pending-review',
    2: 'status-published',
    3: 'status-disabled',
  }
  return map[status] || ''
}

/** 文档处理状态文本 */
export function processText(status: number): string {
  const map: Record<number, string> = {
    0: '待处理',
    1: '处理中',
    2: '已完成',
    3: '失败',
  }
  return map[status] || '未知'
}

/** 文档处理状态 CSS 类 */
export function processClass(status: number): string {
  const map: Record<number, string> = {
    0: 'process-pending',
    1: 'process-processing',
    2: 'process-done',
    3: 'process-failed',
  }
  return map[status] || ''
}
