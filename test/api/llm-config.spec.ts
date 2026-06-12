import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertFields,
  apiUrl, uniqueName,
} from '../utils/test-helpers.js';

/**
 * LLM 配置接口集成测试 — 全覆盖 6 个端点。
 */

const token = getToken();

test.describe('GET /api/v1/admin/llm-configs', () => {
  test('返回配置列表，含 embedding_model/vector_dimension 等完整字段', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);
    if (data.length > 0) {
      assertFields(data[0], {
        id: 'number', name: 'string', provider_type: 'number',
        base_url: 'string', llm_model: 'string', embedding_model: 'string',
        max_tokens: 'number', vector_dimension: 'number', is_default: 'boolean',
        api_key_masked: 'string',
      });
    }
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/admin/llm-configs'));
    await assertError(resp, 401, 10001);
  });
});

test.describe('POST /api/v1/admin/llm-configs — 创建配置完整生命周期', () => {
  let configId: number;

  test('创建 llama.cpp 配置成功', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const name = uniqueName('llama-cpp-test');
    const resp = await request.post(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
      data: {
        name, provider_type: 1,
        base_url: 'http://llama-cpp:8080/v1', api_key: '',
        llm_model: 'qwen3-4b', embedding_model: 'bge-m3',
        max_tokens: 8192, vector_dimension: 1024, is_default: false,
      },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBeGreaterThan(0);
    configId = data.id as number;
  });

  test('创建 OpenAI-compatible 配置 + API Key 掩码验证', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const name = uniqueName('openai-test');
    const resp = await request.post(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
      data: {
        name, provider_type: 2,
        base_url: 'https://api.openai.com/v1', api_key: 'sk-test-key-1234567890',
        llm_model: 'gpt-4o-mini', embedding_model: 'text-embedding-3-small',
        max_tokens: 16384, vector_dimension: 1536, is_default: false,
      },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBeGreaterThan(0);

    // 验证 API Key 掩码
    const detailResp = await request.get(apiUrl(`/api/v1/admin/llm-configs/${data.id}`), {
      headers: authHeaders(token),
    });
    const detailBody = await detailResp.json();
    if (detailBody.code === 0) {
      const masked = detailBody.data.api_key_masked as string;
      if (masked) expect(masked).toContain('****');
    }

    // 清理
    await request.delete(apiUrl(`/api/v1/admin/llm-configs/${data.id}`), {
      headers: authHeaders(token),
    });
  });

  test('缺少必填字段返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
      data: { name: '不完整' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('无效 provider_type (99) 返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
      data: { name: '无效类型', provider_type: 99, base_url: 'http://localhost/v1', llm_model: 't', embedding_model: 't', max_tokens: 100, vector_dimension: 10 },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('查看配置详情', async ({ request }) => {
    if (!token || !configId) { test.skip(true, '缺少 token 或配置'); return; }
    const resp = await request.get(apiUrl(`/api/v1/admin/llm-configs/${configId}`), {
      headers: authHeaders(token),
    });
    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBe(configId);
  });

  test('更新配置成功', async ({ request }) => {
    if (!token || !configId) { test.skip(true, '缺少 token 或配置'); return; }
    const resp = await request.put(apiUrl(`/api/v1/admin/llm-configs/${configId}`), {
      headers: authHeaders(token),
      data: {
        name: `${uniqueName('updated')}`, provider_type: 1,
        base_url: 'http://llama-cpp:8080/v1', api_key: '',
        llm_model: 'qwen3-4b', embedding_model: 'bge-m3',
        max_tokens: 16384, vector_dimension: 1024, is_default: false,
      },
    });
    await assertSuccess(resp);
  });

  test('不存在的配置返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/llm-configs/99999'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });

  test.afterAll(async ({ request }) => {
    if (configId && token) {
      await request.delete(apiUrl(`/api/v1/admin/llm-configs/${configId}`), {
        headers: authHeaders(token),
      });
    }
  });
});

test.describe('DELETE /api/v1/admin/llm-configs/:id', () => {
  test('不能删除默认配置', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const listResp = await request.get(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
    });
    const listBody = await listResp.json();
    if (listBody.code !== 0 || !listBody.data?.length) { test.skip(true, '无配置'); return; }
    const defaultCfg = listBody.data.find((c: Record<string, unknown>) => c.is_default === true);
    if (!defaultCfg) { test.skip(true, '无默认配置'); return; }

    const resp = await request.delete(apiUrl(`/api/v1/admin/llm-configs/${defaultCfg.id}`), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 400], 10003);
  });
});

test.describe('POST /api/v1/admin/llm-configs/:id/test', () => {
  test('测试连接返回 success + latency_ms（连接可能失败）', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const listResp = await request.get(apiUrl('/api/v1/admin/llm-configs'), {
      headers: authHeaders(token),
    });
    const listBody = await listResp.json();
    if (listBody.code !== 0 || !listBody.data?.length) { test.skip(true, '无配置'); return; }
    const configId = listBody.data[0].id;

    const resp = await request.post(apiUrl(`/api/v1/admin/llm-configs/${configId}/test`), {
      headers: authHeaders(token),
    });

    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code).toBe(0);
    const data = body.data as Record<string, unknown>;
    assertFields(data, { success: 'boolean', latency_ms: 'number' });
    if (data.success) {
      expect(data.model).toBeDefined();
    } else {
      expect(data.error).toBeDefined();
    }
  });
});
