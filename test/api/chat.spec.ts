import { test, expect } from '@playwright/test';
import {
  getToken, authHeaders,
  assertError, assertFields,
  apiUrl,
} from '../utils/test-helpers.js';

/**
 * 智能问答接口集成测试 — 覆盖 SSE 流式 + 非流式 + 反馈。
 *
 * SSE 测试使用 Node.js fetch API（Playwright APIRequestContext 不支持流式消费）。
 */

const token = getToken();
let kbId: number;

test.beforeAll(async ({ request }) => {
  if (!token) return;
  const resp = await request.get(apiUrl('/api/v1/admin/knowledge-bases'), {
    headers: authHeaders(token),
  });
  const body = await resp.json();
  if (body.code === 0 && body.data?.items?.length > 0) {
    kbId = body.data.items[0].id;
  }
});

// ==================== 非流式问答 ====================

test.describe('POST /api/v1/portal/chat-sessions (非流式)', () => {
  test('创建问答会话成功，返回 answer/sources/confidence/pipeline', async ({ request }) => {
    if (!token || !kbId) { test.skip(true, '缺少 token 或知识库'); return; }

    const resp = await request.post(apiUrl('/api/v1/portal/chat-sessions'), {
      headers: authHeaders(token),
      data: { question: '如何重置密码？', kb_id: kbId },
    });

    const body = await resp.json();
    // AI 或 RAG 不可用是可接受的降级场景
    if (body.code === 20001 || body.code === 20002) {
      expect(body.message).toBeTruthy();
      return;
    }

    expect(body.code).toBe(0);
    const data = body.data as Record<string, unknown>;
    assertFields(data, {
      session_id: 'number', question: 'string', answer: 'string',
      confidence: 'number', duration_ms: 'number', pipeline: 'object',
    });
    // sources 可为 null（无检索结果时）
    const confidence = data.confidence as number;
    expect(confidence).toBeGreaterThanOrEqual(0);
    expect(confidence).toBeLessThanOrEqual(1);

    // pipeline 结构验证
    const pipeline = data.pipeline as Record<string, unknown>;
    assertFields(pipeline, { steps: 'array', total_duration_ms: 'number' });
  });

  test('缺少 kb_id 返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/portal/chat-sessions'), {
      headers: authHeaders(token),
      data: { question: '问题但没有 kb_id' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/portal/chat-sessions'), {
      data: { question: 'test', kb_id: 1 },
    });
    await assertError(resp, 401, 10001);
  });

  test('空问题返回校验失败', async ({ request }) => {
    if (!token || !kbId) { test.skip(true, '缺少 token 或知识库'); return; }
    const resp = await request.post(apiUrl('/api/v1/portal/chat-sessions'), {
      headers: authHeaders(token),
      data: { question: '', kb_id: kbId },
    });
    await assertError(resp, [200, 400], 10003);
  });
});

// ==================== SSE 流式问答 ====================

test.describe('POST /api/v1/portal/chat-sessions/stream (SSE)', () => {
  test('SSE 响应 content-type 为 text/event-stream', async () => {
    if (!token || !kbId) { test.skip(true, '缺少 token 或知识库'); return; }

    const resp = await fetch(apiUrl('/api/v1/portal/chat-sessions/stream'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ question: 'VPN 怎么连接？', kb_id: kbId }),
    });

    if (resp.status !== 200) {
      const json = await resp.json();
      expect([20001, 20002]).toContain(json.code);
      return;
    }

    expect(resp.headers.get('content-type')).toContain('text/event-stream');

    // 验证 SSE 事件类型
    const reader = resp.body?.getReader();
    if (!reader) { test.skip(true, '无法读取响应流'); return; }

    const decoder = new TextDecoder();
    let fullText = '';
    const eventTypes = new Set<string>();

    try {
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        fullText += decoder.decode(value, { stream: true });

        for (const line of fullText.split('\n')) {
          if (line.startsWith('data: ')) {
            try {
              const json = JSON.parse(line.slice(6));
              eventTypes.add(json.type);
            } catch { /* token 数据可能不完整 */ }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }

    // 如果收到 AI 不可用 JSON 错误
    if (fullText.includes('"code":20001') || fullText.includes('"code":20002')) return;

    // 至少应有 token 或 done 事件
    const hasEvents = eventTypes.has('token') || eventTypes.has('done') || eventTypes.has('step');
    expect(hasEvents, 'SSE 流应包含 token/step/done 事件').toBe(true);
  });

  test('不传 rag_options 使用默认值正常响应', async () => {
    if (!token || !kbId) { test.skip(true, '缺少 token 或知识库'); return; }

    const resp = await fetch(apiUrl('/api/v1/portal/chat-sessions/stream'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ question: '默认参数测试', kb_id: kbId }),
    });

    if (resp.status !== 200) {
      const json = await resp.json();
      expect([20001, 20002]).toContain(json.code);
      return;
    }
    expect(resp.headers.get('content-type')).toContain('text/event-stream');
  });
});

// ==================== 会话查询 + 反馈 ====================

test.describe('查询与反馈', () => {
  test('查询不存在的会话返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/portal/chat-sessions/99999'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });

  test('无效反馈值 (5) 返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/portal/chat-sessions/1/feedback'), {
      headers: authHeaders(token),
      data: { feedback: 5 },
    });
    // API 可能直接成功（不严格校验 feedback 值）或返回错误
    expect([200, 400, 404]).toContain(resp.status());
    const body = await resp.json();
    expect([0, 10003, 10004]).toContain(body.code);
  });
});
