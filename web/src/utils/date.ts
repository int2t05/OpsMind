/**
 * 日期格式化共享工具
 *
 * 从 admin/TicketList, admin/TicketDetail, portal/TicketQuery, portal/TicketDetail
 * 提取重复定义的 formatDate 函数，统一 dateStr 为空时的回退策略。
 */

/**
 * 格式化为 YYYY/MM/DD HH:mm 的中文日期。
 * 优先使用 Intl API 以匹配 Linear Design 日期风格；旧浏览器回退到 substring 截取。
 */
export function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  try {
    return new Date(dateStr).toLocaleDateString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit',
    })
  } catch {
    // 回退：直接截取前 16 个字符 (YYYY-MM-DDTHH:mm)
    return dateStr.substring(0, 16)
  }
}
