/**
 * Token 存取工具函数
 *
 * 操作 localStorage 存储 JWT token。
 * key 固定为 'opsmind_token'，与后端约定一致。
 */

const TOKEN_KEY = 'opsmind_token'
const REFRESH_KEY = 'opsmind_refresh_token'
const USER_KEY = 'opsmind_user'

import type { MenuItem } from '@/types/menu'

interface StoredUserInfo {
  user: { id: number; username: string; real_name: string; phone: string; email: string; first_login: boolean }
  roles: string[]
  permissions: string[]
  menus: MenuItem[]
}

/**
 * 获取存储的 token
 */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}

export function setRefreshToken(token: string): void {
  localStorage.setItem(REFRESH_KEY, token)
}

export function removeRefreshToken(): void {
  localStorage.removeItem(REFRESH_KEY)
}

/**
 * 持久化用户信息快照，用于页面刷新后恢复角色/权限/菜单。
 * 避免路由守卫因 Pinia 状态丢失而误判权限。
 */
export function getUserInfo(): StoredUserInfo | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function setUserInfo(data: StoredUserInfo): void {
  localStorage.setItem(USER_KEY, JSON.stringify(data))
}

export function removeUserInfo(): void {
  localStorage.removeItem(USER_KEY)
}
