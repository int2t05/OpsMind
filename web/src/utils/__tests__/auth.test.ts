import { describe, it, expect, beforeEach } from 'vitest'
import { getToken, setToken, removeToken } from '../auth'

describe('auth utils', () => {
  beforeEach(() => {
    // 每个测试前清除 localStorage
    localStorage.clear()
  })

  describe('getToken', () => {
    it('should return null when no token stored', () => {
      expect(getToken()).toBeNull()
    })

    it('should return stored token', () => {
      const token = 'test-token-123'
      localStorage.setItem('opsmind_token', token)
      expect(getToken()).toBe(token)
    })
  })

  describe('setToken', () => {
    it('should store token in localStorage', () => {
      const token = 'test-token-456'
      setToken(token)
      expect(localStorage.getItem('opsmind_token')).toBe(token)
    })

    it('should overwrite existing token', () => {
      setToken('old-token')
      setToken('new-token')
      expect(getToken()).toBe('new-token')
    })
  })

  describe('removeToken', () => {
    it('should remove token from localStorage', () => {
      setToken('token-to-remove')
      removeToken()
      expect(getToken()).toBeNull()
    })

    it('should not throw when no token exists', () => {
      expect(() => removeToken()).not.toThrow()
    })
  })
})
