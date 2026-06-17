import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

// Mock API 模块
vi.mock('../../api/auth', () => ({
  login: vi.fn(),
  refreshToken: vi.fn(),
  changePassword: vi.fn(),
  logout: vi.fn(),
}))

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  describe('initial state', () => {
    it('should have empty initial state', () => {
      const store = useAuthStore()
      expect(store.token).toBe('')
      expect(store.user).toBeNull()
      expect(store.roles).toEqual([])
      expect(store.permissions).toEqual([])
      expect(store.menus).toEqual([])
    })
  })

  describe('getters', () => {
    it('isLoggedIn should be false when no token', () => {
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(false)
    })

    it('isLoggedIn should be true when token exists', () => {
      const store = useAuthStore()
      store.token = 'test-token'
      expect(store.isLoggedIn).toBe(true)
    })

    it('hasPermission should return true for matching permission', () => {
      const store = useAuthStore()
      store.permissions = ['ticket:read', 'ticket:write']
      expect(store.hasPermission('ticket:read')).toBe(true)
    })

    it('hasPermission should return false for non-matching permission', () => {
      const store = useAuthStore()
      store.permissions = ['ticket:read']
      expect(store.hasPermission('system:config')).toBe(false)
    })

    it('hasPermission should return false when no permissions', () => {
      const store = useAuthStore()
      expect(store.hasPermission('ticket:read')).toBe(false)
    })
  })

  describe('actions', () => {
    it('setToken should update token and localStorage', () => {
      const store = useAuthStore()
      store.setToken('new-token')
      expect(store.token).toBe('new-token')
      expect(localStorage.getItem('opsmind_token')).toBe('new-token')
    })

    it('clearAuth should reset all state', () => {
      const store = useAuthStore()
      store.token = 'test'
      store.permissions = ['test']
      store.clearAuth()
      expect(store.token).toBe('')
      expect(store.permissions).toEqual([])
      expect(localStorage.getItem('opsmind_token')).toBeNull()
    })
  })
})
