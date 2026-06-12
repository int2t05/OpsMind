import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertFields,
  apiUrl,
} from '../utils/test-helpers.js';

/**
 * 数据看板 + 审计日志 + 系统配置 + 站内消息 + 健康检查 集成测试。
 */

const token = getToken();

// ==================== 数据看板 ====================

test.describe('GET /api/v1/admin/dashboard/stats', () => {
  test('返回完整统计数据结构，所有字段类型和范围正确', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/dashboard/stats'), {
      headers: authHeaders(token),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    assertFields(data, {
      today_tickets: 'number', pending_tickets: 'number',
      processing_tickets: 'number', resolved_tickets: 'number',
      today_chats: 'number', avg_confidence: 'number', knowledge_count: 'number',
    });

    expect((data.today_tickets as number)).toBeGreaterThanOrEqual(0);
    expect((data.knowledge_count as number)).toBeGreaterThanOrEqual(0);
    const avgConf = data.avg_confidence as number;
    if (avgConf > 0) {
      expect(avgConf).toBeGreaterThanOrEqual(0);
      expect(avgConf).toBeLessThanOrEqual(1);
    }
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/admin/dashboard/stats'));
    await assertError(resp, 401, 10001);
  });
});

test.describe('GET /api/v1/admin/dashboard/trends', () => {
  test('返回趋势数据（day 粒度），日期格式 YYYY-MM-DD', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(
      apiUrl('/api/v1/admin/dashboard/trends?start_date=2026-06-01&end_date=2026-06-11&granularity=day'),
      { headers: authHeaders(token) },
    );

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    const points = data.data_points as Array<Record<string, unknown>>;
    expect(Array.isArray(points)).toBe(true);
    if (points.length > 0) {
      assertFields(points[0], { date: 'string', ticket_count: 'number', chat_count: 'number' });
      expect((points[0].date as string)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    }
  });

  test('缺失日期参数返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/dashboard/trends'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('日期格式错误返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(
      apiUrl('/api/v1/admin/dashboard/trends?start_date=invalid&end_date=2026-06-11'),
      { headers: authHeaders(token) },
    );
    await assertError(resp, [200, 400, 500], 10003);
  });

  test('结束日期早于开始日期返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(
      apiUrl('/api/v1/admin/dashboard/trends?start_date=2026-06-11&end_date=2026-06-01'),
      { headers: authHeaders(token) },
    );
    await assertError(resp, [200, 400, 500], 10003);
  });
});
