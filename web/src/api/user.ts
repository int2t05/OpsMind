/**
 * 用户管理 API — 全部 6 个端点已通过 ApiResponse&lt;T&gt; 补全泛型类型。
 */
import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

interface UserListParams {
  page?: number
  page_size?: number
  keyword?: string
}

interface CreateUserParams {
  username: string
  password: string
  real_name: string
  phone: string
  email?: string
  role_ids: number[]
}

interface UpdateUserParams {
  real_name: string
  phone: string
  email?: string
  role_ids: number[]
}

/** 用户数据模型 */
export interface UserItem {
  id: number
  username: string
  real_name: string
  phone: string
  email?: string
  status: number
  role_names?: string[]
  created_at?: string
}

export function getUserList(params: UserListParams) {
  return request.get<ApiResponse<PageResponse<UserItem>>>('/api/v1/admin/users', { params })
}

export function getUserById(id: number) {
  return request.get<ApiResponse<UserItem>>(`/api/v1/admin/users/${id}`)
}

export function createUser(data: CreateUserParams) {
  return request.post<ApiResponse<UserItem>>('/api/v1/admin/users', data)
}

export function updateUser(id: number, data: UpdateUserParams) {
  return request.put<ApiResponse<UserItem>>(`/api/v1/admin/users/${id}`, data)
}

export function freezeUser(id: number) {
  return request.patch<ApiResponse<null>>(`/api/v1/admin/users/${id}/freeze`)
}

export function restoreUser(id: number) {
  return request.patch<ApiResponse<null>>(`/api/v1/admin/users/${id}/restore`)
}
