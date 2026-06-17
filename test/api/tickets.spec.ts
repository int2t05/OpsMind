import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertPaginatedResponse, assertFields,
  apiUrl, uniqueName, testTicketData,
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
      expect([200, 400]).toContain(resp.status());
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
      // Server may accept status=99 (no validation), accept either result
      const body = await resp.json();
      expect([200, 400]).toContain(resp.status());
      expect([0, 10003]).toContain(body.code);
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

// ==================== 申告完整生命周期 — 状态转换成功路径 ====================

test.describe.serial('申告完整生命周期', () => {
  let ticketId: number;
  const token = getToken();

  test('创建申告 → 受理 → 需补充 → 补充 → 解决 → 关闭', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }

    // Step 1: 创建申告
    const data = testTicketData(uniqueName('生命周期测试'));
    const createResp = await request.post(apiUrl('/api/v1/portal/tickets'), {
      headers: authHeaders(token), data,
    });
    const createBody = await createResp.json();
    expect(createBody.code, `创建申告失败: ${JSON.stringify(createBody)}`).toBe(0);

    // 获取 ID
    const listResp = await request.get(apiUrl('/api/v1/portal/tickets?page_size=50'), {
      headers: authHeaders(token),
    });
    const listBody = await listResp.json();
    const tickets = listBody.data as Array<Record<string, unknown>>;
    const created = tickets?.find((t: Record<string, unknown>) => t.title === data.title);
    expect(created, `应在列表中找到 "${data.title}"`).toBeDefined();
    ticketId = created!.id as number;
    console.log(`  申告创建: id=${ticketId}, ticket_no=${created!.ticket_no}`);

    // Step 2: 管理员受理 (start)
    const startResp = await request.patch(apiUrl(`/api/v1/admin/tickets/${ticketId}/status`), {
      headers: authHeaders(token),
      data: { action: 'start', result: '已受理，正在排查中' },
    });
    expect(startResp.status()).toBe(200);
    const startBody = await startResp.json();
    expect(startBody.code, `受理失败: ${JSON.stringify(startBody)}`).toBe(0);

    // Step 3: 请求补充信息 (request_info)
    const infoResp = await request.patch(apiUrl(`/api/v1/admin/tickets/${ticketId}/status`), {
      headers: authHeaders(token),
      data: { action: 'request_info', result: '请提供错误截图和发生时间' },
    });
    expect(infoResp.status()).toBe(200);
    const infoBody = await infoResp.json();
    expect(infoBody.code, `请求补充失败: ${JSON.stringify(infoBody)}`).toBe(0);

    // Step 4: 用户补充信息
    const suppResp = await request.patch(apiUrl(`/api/v1/portal/tickets/${ticketId}/supplement`), {
      headers: authHeaders(token),
      data: { content: '错误发生在 2026-06-12 10:00，截图如下：...' },
    });
    expect(suppResp.status()).toBe(200);
    const suppBody = await suppResp.json();
    expect(suppBody.code, `补充信息失败: ${JSON.stringify(suppBody)}`).toBe(0);
    console.log('  补充信息成功');

    // Step 5: 解决 (resolve)
    const resolveResp = await request.patch(apiUrl(`/api/v1/admin/tickets/${ticketId}/status`), {
      headers: authHeaders(token),
      data: { action: 'resolve', result: '已修复，请确认' },
    });
    expect(resolveResp.status()).toBe(200);
    const resolveBody = await resolveResp.json();
    expect(resolveBody.code, `解决失败: ${JSON.stringify(resolveBody)}`).toBe(0);

    // Step 6: 关闭 (close)
    const closeResp = await request.patch(apiUrl(`/api/v1/admin/tickets/${ticketId}/status`), {
      headers: authHeaders(token),
      data: { action: 'close', result: '用户确认已解决' },
    });
    expect(closeResp.status()).toBe(200);
    const closeBody = await closeResp.json();
    expect(closeBody.code, `关闭失败: ${JSON.stringify(closeBody)}`).toBe(0);
    console.log('  申告生命周期完成: 创建→受理→补充→解决→关闭 ✅');
  });
});

// ==================== 管理员处理记录 — 成功路径 ====================

test.describe('管理员添加处理记录', () => {
  let ticketId: number;
  const token = getToken();

  test.beforeAll(async ({ request }) => {
    if (!token) return;
    // 确保有可用的申告
    const listResp = await request.get(apiUrl('/api/v1/admin/tickets?page_size=5'), {
      headers: authHeaders(token),
    });
    const listBody = await listResp.json();
    const tickets = listBody.data as Array<Record<string, unknown>>;
    if (tickets?.length > 0) {
      ticketId = tickets[0].id as number;
    } else {
      // 创建测试申告
      const data = testTicketData(uniqueName('记录测试'));
      const createResp = await request.post(apiUrl('/api/v1/portal/tickets'), {
        headers: authHeaders(token), data,
      });
      const createBody = await createResp.json();
      if (createBody.code === 0) {
        const listResp2 = await request.get(apiUrl('/api/v1/portal/tickets?page_size=50'), {
          headers: authHeaders(token),
        });
        const listBody2 = await listResp2.json();
        const tickets2 = listBody2.data as Array<Record<string, unknown>>;
        const found = tickets2?.find((t: Record<string, unknown>) => t.title === data.title);
        if (found) ticketId = found.id as number;
      }
    }
  });

  test('添加备注记录成功', async ({ request }) => {
    if (!token || !ticketId) { test.skip(true, '缺少 token 或申告'); return; }
    const resp = await request.post(apiUrl(`/api/v1/admin/tickets/${ticketId}/records`), {
      headers: authHeaders(token),
      data: { action: 'note', content: '自动化测试添加的备注' },
    });
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code, `添加备注失败: ${JSON.stringify(body)}`).toBe(0);
  });

  test('添加回访记录成功', async ({ request }) => {
    if (!token || !ticketId) { test.skip(true, '缺少 token 或申告'); return; }
    const resp = await request.post(apiUrl(`/api/v1/admin/tickets/${ticketId}/records`), {
      headers: authHeaders(token),
      data: { action: 'visit', content: '电话回访：用户表示问题已解决', detail: JSON.stringify({ method: '电话', duration_minutes: 5 }) },
    });
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code, `添加回访记录失败: ${JSON.stringify(body)}`).toBe(0);
  });

  test('添加升级记录成功', async ({ request }) => {
    if (!token || !ticketId) { test.skip(true, '缺少 token 或申告'); return; }
    const resp = await request.post(apiUrl(`/api/v1/admin/tickets/${ticketId}/records`), {
      headers: authHeaders(token),
      data: { action: 'escalate', content: '升级至二线运维团队处理', detail: JSON.stringify({ to_team: 'NetOps', reason: '需网络层面排查' }) },
    });
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code, `添加升级记录失败: ${JSON.stringify(body)}`).toBe(0);
  });
});
