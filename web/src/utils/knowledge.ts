/**
 * 知识库共享工具函数
 *
 * 从 admin/KnowledgeList, admin/KnowledgeEdit 提取重复定义的状态显示函数。
 */

/** 文章审核状态 CSS 类名（1-based，与后端 model 对齐） */
export function articleStatusClass(status: number): string {
  const map: Record<number, string> = {
    1: 'draft', 2: 'pending', 3: 'approved', 4: 'published', 5: 'disabled', 6: 'rejected',
  }
  return map[status] || ''
}

/** 文章审核状态显示文本（1-based，与后端 model 对齐） */
export function articleStatusText(status: number): string {
  const map: Record<number, string> = {
    1: '草稿', 2: '待审核', 3: '已通过', 4: '已发布', 5: '已停用', 6: '已驳回',
  }
  return map[status] || '未知'
}

/** 文档处理状态 CSS 类名（后端返回字符串） */
export function processClass(status: string | undefined): string {
  if (!status) return ''
  const map: Record<string, string> = {
    pending: 'pending', parsing: 'pending', chunking: 'pending', embedding: 'pending',
    completed: 'completed', failed: 'failed',
  }
  return map[status] || ''
}

/** 文档处理状态显示文本（后端返回字符串） */
export function processText(status: string | undefined): string {
  if (!status) return '-'
  const map: Record<string, string> = {
    pending: '待处理', parsing: '解析中', chunking: '分块中', embedding: '向量化中',
    completed: '已完成', failed: '失败',
  }
  return map[status] || '-'
}
