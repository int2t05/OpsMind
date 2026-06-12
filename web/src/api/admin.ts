import type { TicketItem, TicketDetail, TicketRecord } from './ticket'
import request from '../utils/request'

/**
 * TODO(api/admin): listAllTickets 的响应类型 `{ data: TicketItem[]; total: number }` 与后端实际结构不一致。
 *                 后端返回格式为 `{ code, message, data: { data: TicketItem[], total } }`（嵌套 data）。
 *                 需要：1) 创建共享的 ApiResponse<T> 类型；2) 修正所有函数的响应泛型。
 * TODO(api/admin): 缺少 getTicketDetail 返回类型中的 records 字段类型 — 应复用 ticket.ts 中的 TicketRecord。
 */

export interface TicketListParams { page?: number; page_size?: number; status?: number }
export interface UpdateStatusParams { action: string; content?: string; operator_id?: number }
export interface AddRecordParams { action: string; content: string }

export function listAllTickets(params?: TicketListParams) {
  return request.get<{ data: TicketItem[]; total: number; page: number; page_size: number }>('/api/v1/admin/tickets', { params })
}

export function getTicketDetail(id: number) {
  return request.get<{ data: TicketDetail }>(`/api/v1/admin/tickets/${id}`)
}

export function updateTicketStatus(id: number, data: UpdateStatusParams) {
  return request.patch(`/api/v1/admin/tickets/${id}/status`, data)
}

export function addTicketRecord(id: number, data: AddRecordParams) {
  return request.post(`/api/v1/admin/tickets/${id}/records`, data)
}

export type { TicketItem, TicketDetail, TicketRecord }
