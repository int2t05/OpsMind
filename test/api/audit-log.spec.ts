import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertPaginatedResponse, assertFields,
  apiUrl,
} from '../utils/test-helpers.js';

/**
 * 审计日志 + 系统配置 + 站内消息 + 健康检查 集成测试 — 全覆盖 7 个端点。
 */

const token = getToken();

// ==================== 审计日志 ====================

test.describe('审计日志', () => {
  test('返回审计日志列表（分页）', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/audit-logs?page=1&page_size=10'), {
      headers: authHeaders(token),
    });

    const body = await assertPaginatedResponse(resp);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);
    if (data.length > 0) {
      assertFields(data[0], {
        id: 'number', operator_id: 'number', operator_name: 'string',
        action: 'string', target_type: 'string', created_at: 'string',
      });
    }
  });

  test('按操作人筛选 (operator_id=1)', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/audit-logs?operator_id=1'), {
      headers: authHeaders(token),
    });
    await assertPaginatedResponse(resp);
  });

  test('按操作类型筛选 (action=knowledge:publish)', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/audit-logs?action=knowledge:publish'), {
      headers: authHeaders(token),
    });
    await assertPaginatedResponse(resp);
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/admin/audit-logs'));
    await assertError(resp, 401, 10001);
  });
});

// ==================== 系统配置 ====================

test.describe('系统配置', () => {
  test('获取 app_name 配置', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/configs/app_name'), {
      headers: authHeaders(token),
    });
    const body = await assertSuccess(resp);
    // data 可能是字符串或对象
    const data = body.data;
    expect(data).toBeTruthy();
  });

  test('不存在的配置键返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/configs/non_existent_key'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });

  test('更新配置后读取验证', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    // 写入
    const putResp = await request.put(apiUrl('/api/v1/admin/configs/app_name'), {
      headers: authHeaders(token),
      data: { value: 'OpsMind Test' },
    });
    expect(putResp.status()).toBe(200);

    // 读取验证
    const getResp = await request.get(apiUrl('/api/v1/admin/configs/app_name'), {
      headers: authHeaders(token),
    });
    const getBody = await getResp.json();
    // data 可能是字符串或对象
    const value = typeof getBody.data === 'string' ? getBody.data : getBody.data?.value;
    expect(value).toBe('OpsMind Test');

    // 恢复
    await request.put(apiUrl('/api/v1/admin/configs/app_name'), {
      headers: authHeaders(token),
      data: { value: 'OpsMind' },
    });
  });
});

// ==================== 站内消息 ====================

test.describe('站内消息', () => {
  test('返回消息列表（分页），含 type 和 is_read', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/portal/messages?page=1&page_size=10'), {
      headers: authHeaders(token),
    });

    const body = await assertPaginatedResponse(resp);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);
    if (data.length > 0) {
      assertFields(data[0], {
        id: 'number', title: 'string', type: 'string', is_read: 'boolean',
      });
    }
  });

  test('返回未读计数', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/portal/messages/unread-count'), {
      headers: authHeaders(token),
    });
    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    assertFields(data, { count: 'number' });
    expect((data.count as number)).toBeGreaterThanOrEqual(0);
  });

  test('不存在的消息标记已读返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.put(apiUrl('/api/v1/portal/messages/99999/read'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });

  test('无 token 访问消息列表返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/portal/messages'));
    await assertError(resp, 401, 10001);
  });
});

// ==================== 健康检查 ====================

test.describe('GET /health', () => {
  test('无需认证，返回 {status: "ok"}', async ({ request }) => {
    const resp = await request.get(apiUrl('/health'));
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.status).toBe('ok');
  });
});
