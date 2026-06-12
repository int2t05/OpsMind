import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'

export interface RoleItem {
  id: number
  name: string
  description: string
  permissions: string[]
  created_at?: string
  updated_at?: string
}

export function getRoleList() {
  return request.get<ApiResponse<RoleItem[]>>('/api/v1/admin/roles')
}
