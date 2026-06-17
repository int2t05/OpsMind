/**
 * 审计日志 API 封装（后台管理端）
 */
import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

export interface AuditLogItem {
  id: number
  username?: string
  operator_name?: string
  action: string
  target_type?: string
  target_id?: number
  detail?: string
  ip_address?: string
  created_at?: string
}

export interface AuditLogListParams {
  page?: number
  page_size?: number
}

export function listAuditLogs(params?: AuditLogListParams) {
  return request.get<ApiResponse<PageResponse<AuditLogItem>>>('/api/v1/admin/audit-logs', { params })
}
