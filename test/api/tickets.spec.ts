import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertPaginatedResponse, assertFields,
  apiUrl, testTicketData,
} from '../utils/test-helpers.js';

/**
 * 申告管理接口集成测试 — 全覆盖门户端 4 端点 + 后台管理 4 端点。
 */

// ==================== 门户端 ====================

test.describe('门户端申告接口', () => {
  let ticketId: number;
  const token = getToken();

  test.describe('POST /api/v1/portal/tickets', () => {
    test('创建申告成功，返回 id', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const data = testTicketData();
      const resp = await request.post(apiUrl('/api/v1/portal/tickets'), {
        headers: authHeaders(token),
        data,
      });

      const body = await resp.json();
      expect(body.code, `创建申告失败: ${JSON.stringify(body)}`).toBe(0);

      // 从列表获取申告 ID（创建可能返回 data: null）
      const listResp = await request.get(apiUrl('/api/v1/portal/tickets?page_size=50'), {
        headers: authHeaders(token),
      });
      const listBody = await listResp.json();
      const tickets = listBody.data as Array<Record<string, unknown>>;
      const created = tickets?.find((t: Record<string, unknown>) => t.title === data.title);
      if (created) ticketId = created.id as number;
    });

    test('缺少必填字段 (description) 返回校验失败', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.post(apiUrl('/api/v1/portal/tickets'), {
        headers: authHeaders(token),
        data: { title: '只有标题' },
      });
      await assertError(resp, [200, 400], 10003);
    });

    test('无效 urgency 值 (99) 返回校验失败', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.post(apiUrl('/api/v1/portal/tickets'), {
        headers: authHeaders(token),
        data: { title: '测试', description: '测试', urgency: 99, contact_phone: '13800000001' },
      });
      await assertError(resp, [200, 400], 10003);
    });

    test('无 token 创建返回 401', async ({ request }) => {
      const resp = await request.post(apiUrl('/api/v1/portal/tickets'), {
        data: { title: 'test', description: 'test', urgency: 1, contact_phone: '13800000001' },
      });
      await assertError(resp, 401, 10001);
    });
  });

  test.describe('GET /api/v1/portal/tickets', () => {
    test('返回我的申告列表（分页），含 ticket_no', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/portal/tickets?page=1&page_size=10'), {
        headers: authHeaders(token),
      });

      const body = await assertPaginatedResponse(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);
      if (data.length > 0) {
        assertFields(data[0], {
          id: 'number', ticket_no: 'string', title: 'string',
          urgency: 'number', status: 'number', status_text: 'string',
        });
      }
    });
  });

  test.describe('GET /api/v1/portal/tickets/:id', () => {
    test('不存在的申告返回 404', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/portal/tickets/99999'), {
        headers: authHeaders(token),
      });
      await assertError(resp, [200, 404], 10004);
    });
  });

  test.describe('PATCH /api/v1/portal/tickets/:id/supplement', () => {
    test('非「需补充信息」状态补充信息应失败', async ({ request }) => {
      if (!token || !ticketId) { test.skip(true, '缺少 token 或申告'); return; }
      const resp = await request.patch(apiUrl(`/api/v1/portal/tickets/${ticketId}/supplement`), {
        headers: authHeaders(token),
        data: { content: '补充信息...' },
      });
      expect(resp.status()).toBe(200);
      const body = await resp.json();
      expect(body.code).toBeGreaterThan(0);
    });
  });
});

// ==================== 后台管理 ====================

test.describe('后台管理申告接口', () => {
  const token = getToken();

  test.describe('GET /api/v1/admin/tickets', () => {
    test('返回全部申告列表（分页）', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/admin/tickets?page=1&page_size=10'), {
        headers: authHeaders(token),
      });
      await assertPaginatedResponse(resp);
    });

    test('按状态筛选 (status=1 待处理)', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/admin/tickets?status=1'), {
        headers: authHeaders(token),
      });
      await assertPaginatedResponse(resp);
    });

    test('按紧急程度筛选 (urgency=2 紧急)', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/admin/tickets?urgency=2'), {
        headers: authHeaders(token),
      });
      await assertPaginatedResponse(resp);
    });

    test('无效 status 值返回校验失败', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.get(apiUrl('/api/v1/admin/tickets?status=99'), {
        headers: authHeaders(token),
      });
      await assertError(resp, [200, 400], 10003);
    });
  });

  test.describe('PATCH /api/v1/admin/tickets/:id/status', () => {
    test('无效 action 返回校验失败', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.patch(apiUrl('/api/v1/admin/tickets/1/status'), {
        headers: authHeaders(token),
        data: { action: 'invalid_action' },
      });
      await assertError(resp, [200, 400], 10003);
    });

    test('不存在的申告返回 404', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.patch(apiUrl('/api/v1/admin/tickets/99999/status'), {
        headers: authHeaders(token),
        data: { action: 'start', result: '测试' },
      });
      await assertError(resp, [200, 404], 10004);
    });
  });

  test.describe('POST /api/v1/admin/tickets/:id/records', () => {
    test('缺少 action 返回校验失败', async ({ request }) => {
      if (!token) { test.skip(true, '缺少 token'); return; }
      const resp = await request.post(apiUrl('/api/v1/admin/tickets/1/records'), {
        headers: authHeaders(token),
        data: { content: '没有 action' },
      });
      await assertError(resp, [200, 400], 10003);
    });
  });
});

// ==================== 权限验证 ====================

test.describe('权限验证', () => {
  test('无 token 访问后台申告列表返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/admin/tickets'));
    await assertError(resp, 401, 10001);
  });
});
