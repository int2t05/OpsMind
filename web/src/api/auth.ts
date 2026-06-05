import request from '../utils/request'

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
  return request.post<LoginResponse>('/api/v1/auth/login', data)
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
