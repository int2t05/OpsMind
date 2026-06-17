import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '../app'

describe('app store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('initial state', () => {
    it('should have sidebar not collapsed by default', () => {
      const store = useAppStore()
      expect(store.sidebarCollapsed).toBe(false)
    })

    it('should have zero unread messages', () => {
      const store = useAppStore()
      expect(store.unreadMessageCount).toBe(0)
    })
  })

  describe('actions', () => {
    it('toggleSidebar should toggle collapsed state', () => {
      const store = useAppStore()
      expect(store.sidebarCollapsed).toBe(false)
      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(true)
      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(false)
    })

    it('setUnreadCount should update count', () => {
      const store = useAppStore()
      store.setUnreadCount(5)
      expect(store.unreadMessageCount).toBe(5)
    })
  })
})
