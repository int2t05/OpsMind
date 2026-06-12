/**
 * 申告 API 封装（门户端）
 *
 * 提供申告创建、列表查询、详情查询、补充信息等门户端接口。
 * 后台管理端申告接口（列表管理、状态操作）在 admin API 中独立封装。
 *
 * TODO(api/ticket): TicketRecord.detail 使用 Record<string, any> — any 应替换为具体类型。
 * TODO(api/ticket): listMyTickets/getTicketDetail 响应类型缺少 ApiResponse<T> 包装，
 *                   TicketListResponse/TicketDetail 无法正确匹配后端 { code, message, data } 结构。
 * TODO(api/ticket): createTicket/supplementTicket 无泛型参数，返回 any — 应补充类型。
 */
import request from '../utils/request'

// =============================================================================
// 类型定义
// =============================================================================

export interface CreateTicketParams {
  title: string
  description: string
  urgency: number          // 1=低, 2=中, 3=高
  impact_scope?: number    // 1=个人, 2=部门, 3=全公司
  affected_systems?: string[]
  contact_phone: string
  contact_email?: string
  chat_context?: string    // JSON 字符串，从问答转申告时带入
}

export interface SupplementTicketParams {
  content: string
}

export interface TicketItem {
  id: number
  ticket_no: string
  user_id: number
  submitter_name: string
  title: string
  urgency: number
  impact_scope: number
  contact_phone: string
  status: number
  status_text: string
  supplement_count: number
  created_at: string
  updated_at: string
}

export interface TicketRecord {
  id: number
  ticket_id: number
  operator_id: number
  action: string
  content: string
  detail?: Record<string, any>
  created_at: string
}

export interface TicketDetail {
  id: number
  ticket_no: string
  user_id: number
  submitter_name: string
  title: string
  description: string
  urgency: number
  impact_scope: number
  affected_systems: string[]
  contact_phone: string
  contact_email: string
  status: number
  status_text: string
  supplement_count: number
  chat_context?: string
  source: number
  records: TicketRecord[]
  created_at: string
  updated_at: string
}

export interface TicketListResponse {
  items: TicketItem[]
  total: number
}

// =============================================================================
// API 方法
// =============================================================================

/** 创建申告 */
export function createTicket(data: CreateTicketParams) {
  return request.post('/api/v1/portal/tickets', data)
}

/** 查询当前用户的申告列表 */
export function listMyTickets(page: number = 1, pageSize: number = 10) {
  return request.get<TicketListResponse>('/api/v1/portal/tickets', {
    params: { page, page_size: pageSize }
  })
}

/** 查询申告详情 */
export function getTicketDetail(id: number) {
  return request.get<TicketDetail>(`/api/v1/portal/tickets/${id}`)
}

/** 补充申告信息（仅"需补充信息"状态可操作） */
export function supplementTicket(id: number, data: SupplementTicketParams) {
  return request.post(`/api/v1/portal/tickets/${id}/supplement`, data)
}
