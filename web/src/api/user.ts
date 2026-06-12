/**
 * TODO(api/user): 所有函数均未传入泛型类型参数，返回值全部为 any。
 *                需定义 ApiResponse<T> 共享类型并为每个函数添加泛型声明，
 *                例如：request.get<ApiResponse<UserListData>>(...)。
 *                同时应将导入路径统一为 @/utils/request（混用相对路径）。
 */
import request from '../utils/request'

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

export function getUserList(params: UserListParams) {
  return request.get('/api/v1/admin/users', { params })
}

export function getUserById(id: number) {
  return request.get(`/api/v1/admin/users/${id}`)
}

export function createUser(data: CreateUserParams) {
  return request.post('/api/v1/admin/users', data)
}

export function updateUser(id: number, data: UpdateUserParams) {
  return request.put(`/api/v1/admin/users/${id}`, data)
}

export function freezeUser(id: number) {
  return request.patch(`/api/v1/admin/users/${id}/freeze`)
}

export function restoreUser(id: number) {
  return request.patch(`/api/v1/admin/users/${id}/restore`)
}
