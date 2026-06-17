import { describe, it, expect, vi } from 'vitest'

// Mock request 模块，拦截 API 调用
vi.mock('../../utils/request', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
  }
}))

import request from '../../utils/request'
import { createTicket, listMyTickets, getTicketDetail, supplementTicket } from '../ticket'

describe('ticket API', () => {
  describe('createTicket', () => {
    it('should POST to portal tickets with correct body', async () => {
      const mockData = {
        title: '网络故障',
        description: '无法上网',
        urgency: 2,
        contact_phone: '13800000001',
      }
      ;(request.post as any).mockResolvedValue({ code: 0, data: null })

      await createTicket(mockData)

      expect(request.post).toHaveBeenCalledWith('/api/v1/portal/tickets', mockData)
    })
  })

  describe('listMyTickets', () => {
    it('should GET portal tickets with pagination params', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { items: [], total: 0 } })

      await listMyTickets(2, 20)

      expect(request.get).toHaveBeenCalledWith('/api/v1/portal/tickets', {
        params: { page: 2, page_size: 20 }
      })
    })

    it('should use default page=1, pageSize=10 when not specified', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { items: [], total: 0 } })

      await listMyTickets()

      expect(request.get).toHaveBeenCalledWith('/api/v1/portal/tickets', {
        params: { page: 1, page_size: 10 }
      })
    })
  })

  describe('getTicketDetail', () => {
    it('should GET portal ticket by id', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: {} })

      await getTicketDetail(42)

      expect(request.get).toHaveBeenCalledWith('/api/v1/portal/tickets/42')
    })
  })

  describe('supplementTicket', () => {
    it('should PATCH supplement with content', async () => {
      ;(request.patch as any).mockResolvedValue({ code: 0, data: null })

      await supplementTicket(42, { content: '补充说明' })

      expect(request.patch).toHaveBeenCalledWith('/api/v1/portal/tickets/42/supplement', {
        content: '补充说明'
      })
    })
  })
})
