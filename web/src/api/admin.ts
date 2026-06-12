import type { TicketItem, TicketDetail, TicketRecord } from './ticket'
import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

export interface TicketListParams { page?: number; page_size?: number; status?: number }
export interface UpdateStatusParams { action: string; result?: string }
export interface AddRecordParams { action: string; content: string }

export function listAllTickets(params?: TicketListParams) {
  return request.get<ApiResponse<PageResponse<TicketItem>>>('/api/v1/admin/tickets', { params })
}

export function getTicketDetail(id: number) {
  return request.get<ApiResponse<TicketDetail>>(`/api/v1/admin/tickets/${id}`)
}

export function updateTicketStatus(id: number, data: UpdateStatusParams) {
  return request.patch<ApiResponse<null>>(`/api/v1/admin/tickets/${id}/status`, data)
}

export function addTicketRecord(id: number, data: AddRecordParams) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/tickets/${id}/records`, data)
}

export type { TicketItem, TicketDetail, TicketRecord }
