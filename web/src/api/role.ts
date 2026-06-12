import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

export interface RoleItem {
  id: number
  name: string
  description: string
  permissions: string[]
  created_at?: string
  updated_at?: string
}

export interface MenuItem {
  id: number
  name: string
  path: string
  icon: string
  parent_id: number
  sort_order: number
  type: number
  children?: MenuItem[]
}

// ==================== 角色 CRUD ====================

export function getRoleList() {
  return request.get<ApiResponse<PageResponse<RoleItem>>>('/api/v1/admin/roles')
}

export function getRoleById(id: number) {
  return request.get<ApiResponse<RoleItem>>(`/api/v1/admin/roles/${id}`)
}

export function createRole(data: { name: string; description?: string; permissions: string[] }) {
  return request.post<ApiResponse<RoleItem>>('/api/v1/admin/roles', data)
}

export function updateRole(id: number, data: { name?: string; description?: string; permissions?: string[] }) {
  return request.put<ApiResponse<RoleItem>>(`/api/v1/admin/roles/${id}`, data)
}

export function deleteRole(id: number) {
  return request.delete<ApiResponse<null>>(`/api/v1/admin/roles/${id}`)
}

// ==================== 菜单管理 ====================

export function listMenus() {
  return request.get<ApiResponse<MenuItem[]>>('/api/v1/admin/menus')
}

export function updateRoleMenus(roleId: number, menuIds: number[]) {
  return request.put<ApiResponse<null>>(`/api/v1/admin/roles/${roleId}/menus`, { menu_ids: menuIds })
}
