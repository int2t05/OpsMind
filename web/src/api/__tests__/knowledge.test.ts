import { describe, it, expect, vi } from 'vitest'

vi.mock('../../utils/request', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  }
}))

import request from '../../utils/request'
import {
  uploadDocuments,
  getDocumentStatus,
  retryDocument,
} from '../knowledge'

describe('knowledge API', () => {
  describe('uploadDocuments', () => {
    it('should POST multipart form to upload documents', async () => {
      ;(request.post as any).mockResolvedValue({ code: 0, data: { articles: [{ id: 1 }] } })

      const formData = new FormData()
      const file = new File(['test content'], 'test.md', { type: 'text/markdown' })
      formData.append('files', file)

      await uploadDocuments(1, formData)

      expect(request.post).toHaveBeenCalledWith(
        '/api/v1/admin/knowledge-bases/1/documents/upload',
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
    })
  })

  describe('getDocumentStatus', () => {
    it('should GET document processing status', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { process_status: 2, process_error: '' } })

      await getDocumentStatus(1, 10)

      expect(request.get).toHaveBeenCalledWith('/api/v1/admin/knowledge-bases/1/documents/10/status')
    })
  })

  describe('retryDocument', () => {
    it('should POST to retry document processing', async () => {
      ;(request.post as any).mockResolvedValue({ code: 0, data: null })

      await retryDocument(1, 10)

      expect(request.post).toHaveBeenCalledWith('/api/v1/admin/knowledge-bases/1/documents/10/retry')
    })
  })
})
