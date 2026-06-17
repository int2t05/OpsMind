import { describe, it, expect, vi } from 'vitest'

vi.mock('../../utils/request', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }
}))

import request from '../../utils/request'
import {
  getLLMConfigs,
  createLLMConfig,
  getLLMConfig,
  updateLLMConfig,
  deleteLLMConfig,
  testLLMConnection,
} from '../llm_config'

describe('llm_config API', () => {
  describe('getLLMConfigs', () => {
    it('should GET list of LLM configs', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: [] })

      await getLLMConfigs()

      expect(request.get).toHaveBeenCalledWith('/api/v1/admin/llm-configs')
    })
  })

  describe('createLLMConfig', () => {
    it('should POST to create a config', async () => {
      const params = {
        name: 'llama.cpp',
        provider_type: 1,
        base_url: 'http://localhost:8080/v1',
        api_key: '',
        llm_model: 'qwen3-4b',
        embedding_model: 'bge-m3',
        max_tokens: 8192,
        vector_dimension: 1024,
        is_default: true,
      }
      ;(request.post as any).mockResolvedValue({ code: 0, data: null })

      await createLLMConfig(params)

      expect(request.post).toHaveBeenCalledWith('/api/v1/admin/llm-configs', params)
    })
  })

  describe('getLLMConfig', () => {
    it('should GET a single config by ID', async () => {
      ;(request.get as any).mockResolvedValue({ code: 0, data: { id: 1, name: 'test' } })

      await getLLMConfig(1)

      expect(request.get).toHaveBeenCalledWith('/api/v1/admin/llm-configs/1')
    })
  })

  describe('updateLLMConfig', () => {
    it('should PUT to update a config', async () => {
      const params = {
        name: 'updated',
        provider_type: 2,
        base_url: 'https://api.openai.com/v1',
        api_key: 'sk-xxx',
        llm_model: 'gpt-4o',
        embedding_model: 'text-embedding-3-small',
        max_tokens: 4096,
        vector_dimension: 1536,
        is_default: false,
      }
      ;(request.put as any).mockResolvedValue({ code: 0, data: null })

      await updateLLMConfig(1, params)

      expect(request.put).toHaveBeenCalledWith('/api/v1/admin/llm-configs/1', params)
    })
  })

  describe('deleteLLMConfig', () => {
    it('should DELETE a config by ID', async () => {
      ;(request.delete as any).mockResolvedValue({ code: 0, data: null })

      await deleteLLMConfig(1)

      expect(request.delete).toHaveBeenCalledWith('/api/v1/admin/llm-configs/1')
    })
  })

  describe('testLLMConnection', () => {
    it('should POST to test connection', async () => {
      ;(request.post as any).mockResolvedValue({ code: 0, data: { success: true, model: 'qwen3-4b', latency: 150 } })

      await testLLMConnection(1)

      expect(request.post).toHaveBeenCalledWith('/api/v1/admin/llm-configs/1/test')
    })
  })
})
