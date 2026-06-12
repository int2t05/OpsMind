/**
 * 申告相关共享工具函数
 *
 * 从 admin/TicketDetail、admin/TicketList、portal/TicketQuery、portal/TicketDetail
 * 四个视图中提取，消除 urgencyText / statusClass 的 4 处重复定义。
 */

/** 申告紧急程度映射 */
export function urgencyText(urgency: number): string {
  const map: Record<number, string> = { 1: '低', 2: '中', 3: '高' }
  return map[urgency] || '未知'
}

/** 申告状态 CSS 类 */
export function ticketStatusClass(status: number): string {
  const map: Record<number, string> = {
    0: 'status-pending',
    1: 'status-processing',
    2: 'status-supplement',
    3: 'status-resolved',
    4: 'status-closed',
  }
  return map[status] || ''
}
