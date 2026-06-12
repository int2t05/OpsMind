import request from '../utils/request'

// 后端统一响应格式（axios 响应拦截器已提取 response.data）
// request.post<T> 中的 T 对应 response.data 的类型
//
// TODO(api/auth): ApiResponse<T> 与 api/dashboard.ts 中重复定义 — 应抽取到 src/types/api.ts 共享。
// TODO(api/auth): LoginResponse.menus 使用 any[] — 应使用 auth store 中已有的 MenuItem 类型。
// TODO(api/auth): refreshToken() 无泛型参数，返回 any — 应补充类型声明。
interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

interface LoginParams {
  username: string
  password: string
}

interface LoginResponse {
  access_token: string
  refresh_token: string
  user: {
    id: number
    username: string
    real_name: string
    phone: string
    email: string
    first_login: boolean
  }
  roles: string[]
  permissions: string[]
  menus: any[]
}

interface ChangePasswordParams {
  old_password: string
  new_password: string
}

export function login(data: LoginParams) {
  return request.post<ApiResponse<LoginResponse>>('/api/v1/auth/login', data)
}

export function refreshToken(refresh_token: string) {
  return request.post('/api/v1/auth/refresh', { refresh_token })
}

export function changePassword(data: ChangePasswordParams) {
  return request.post('/api/v1/auth/change-password', data)
}

export function logout() {
  return request.post('/api/v1/auth/logout')
}
