/**
 * 申告共享工具函数
 *
 * 从 admin/TicketList, admin/TicketDetail, portal/TicketQuery, portal/TicketDetail 4 个视图
 * 提取重复定义的辅助函数，统一维护。
 */

/** 紧急程度显示文本 */
export function urgencyText(urgency: number): string {
  const map: Record<number, string> = { 1: '低', 2: '中', 3: '高' }
  return map[urgency] || '未知'
}

/** 紧急程度 CSS 类名 */
export function urgencyClass(urgency: number): string {
  if (urgency === 3) return 'high'
  if (urgency === 2) return 'medium'
  return 'low'
}

/** 申告状态 CSS 类名 */
export function ticketStatusClass(status: number): string {
  if (status === 1) return 'pending'
  if (status === 2) return 'processing'
  if (status === 3) return 'supplement'
  if (status === 4) return 'resolved'
  return 'closed'
}

/** 影响范围显示文本 */
export function scopeText(scope: number): string {
  const map: Record<number, string> = { 1: '个人', 2: '部门', 3: '全公司' }
  return map[scope] || '未知'
}

/** 处理记录操作显示文本（合并 admin + portal 差异映射） */
export function actionText(action: string): string {
  const map: Record<string, string> = {
    create: '创建申告',
    start: '开始处理',
    request_info: '需补充信息',
    supplement: '补充信息',
    resolve: '已解决',
    close: '关闭',
    remark: '备注',
  }
  return map[action] || action
}
