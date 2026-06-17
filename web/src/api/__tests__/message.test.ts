import { describe, it, expect, vi } from 'vitest'

vi.mock('../../utils/request', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
  }
}))

import request from '../../utils/request'
import { listMessages, markAsRead, getUnreadCount } from '../message'

describe('message API', () => {
  describe('listMessages', () => {
    it('should GET portal messages with pagination', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { items: [], total: 0 } })

      await listMessages(1, 10)

      expect(request.get).toHaveBeenCalledWith('/api/v1/portal/messages', {
        params: { page: 1, page_size: 10 }
      })
    })
  })

  describe('markAsRead', () => {
    it('should PUT to mark message as read', async () => {
      ;(request.put as any).mockResolvedValue({ code: 0, data: null })

      await markAsRead(7)

      expect(request.put).toHaveBeenCalledWith('/api/v1/portal/messages/7/read')
    })
  })

  describe('getUnreadCount', () => {
    it('should GET unread count', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { count: 3 } })

      await getUnreadCount()

      expect(request.get).toHaveBeenCalledWith('/api/v1/portal/messages/unread-count')
    })
  })
})
