/**
 * 申告相关类型定义
 *
 * 提供状态枚举映射，替代各视图中分散的魔数。
 */

/** 申告紧急程度 */
export enum TicketUrgency {
  Low = 1,
  Medium = 2,
  High = 3,
}

/** 申告状态 */
export enum TicketStatus {
  Pending = 1,
  Processing = 2,
  NeedSupplement = 3,
  Resolved = 4,
  Closed = 5,
}

/** 申告状态文本映射 */
export const TICKET_STATUS_TEXT: Record<number, string> = {
  [TicketStatus.Pending]: '待处理',
  [TicketStatus.Processing]: '处理中',
  [TicketStatus.NeedSupplement]: '需补充信息',
  [TicketStatus.Resolved]: '已解决',
  [TicketStatus.Closed]: '已关闭',
}

/** 申告紧急程度文本映射 */
export const TICKET_URGENCY_TEXT: Record<number, string> = {
  [TicketUrgency.Low]: '低',
  [TicketUrgency.Medium]: '中',
  [TicketUrgency.High]: '高',
}
