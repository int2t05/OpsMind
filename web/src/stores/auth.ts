import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getToken, setToken as saveToken, removeToken, setRefreshToken as saveRefresh, removeRefreshToken, getUserInfo, setUserInfo as saveUserInfo, removeUserInfo } from '../utils/auth'
import type { MenuItem } from '@/types/menu'

interface UserInfo {
  id: number
  username: string
  real_name: string
  phone: string
  email: string
  status?: number
  first_login: boolean
}

export type { MenuItem }

export const useAuthStore = defineStore('auth', () => {
  // 从 localStorage 恢复用户权限快照，避免刷新后路由守卫因空角色拦截
  const stored = getUserInfo()
  const token = ref(getToken() || '')
  const user = ref<UserInfo | null>(stored?.user || null)
  const roles = ref<string[]>(stored?.roles || [])
  const permissions = ref<string[]>(stored?.permissions || [])
  const menus = ref<MenuItem[]>(stored?.menus || [])

  const isLoggedIn = computed(() => !!token.value)

  const hasPermission = (perm: string) => {
    return permissions.value.includes(perm)
  }

  const setToken = (newToken: string, refreshToken?: string) => {
    token.value = newToken
    saveToken(newToken)
    if (refreshToken) saveRefresh(refreshToken)
  }

  const clearAuth = () => {
    token.value = ''
    user.value = null
    roles.value = []
    permissions.value = []
    menus.value = []
    removeToken()
    removeRefreshToken()
    removeUserInfo()
  }

  const setUserInfo = (data: {
    user: UserInfo
    roles: string[]
    permissions: string[]
    menus: MenuItem[]
  }) => {
    user.value = data.user
    roles.value = data.roles
    permissions.value = data.permissions
    menus.value = data.menus
    // 持久化快照到 localStorage，防止刷新后路由守卫误判
    saveUserInfo(data)
  }

  return {
    token, user, roles, permissions, menus,
    isLoggedIn, hasPermission,
    setToken, clearAuth, setUserInfo,
  }
})
