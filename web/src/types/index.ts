/**
 * 共享类型定义统—入口
 *
 * 统一导出所有共享类型，消费方只需 import from '@/types'。
 */

export type { ApiResponse, PageResponse } from './api'
export type { MenuItem } from './menu'
export { TicketUrgency, TicketStatus, TICKET_STATUS_TEXT, TICKET_URGENCY_TEXT } from './ticket'
export { KnowledgeStatus, ProcessStatus, KNOWLEDGE_STATUS_TEXT, PROCESS_STATUS_TEXT } from './knowledge'
