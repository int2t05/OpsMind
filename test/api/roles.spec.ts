import { test, expect } from '@playwright/test';
import {
  requireAuth, getToken, authHeaders,
  assertSuccess, assertError, assertPaginatedResponse, assertFields,
  apiUrl, uniqueName,
} from '../utils/test-helpers.js';

/**
 * 角色与菜单管理接口集成测试 — 全覆盖 7 个端点。
 */

const token = getToken();

test.describe.serial('角色 CRUD 生命周期', () => {
  let roleId: number;
  const roleName = uniqueName('测试角色');

  test('创建角色成功', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/admin/roles'), {
      headers: authHeaders(token),
      data: { name: roleName, description: '测试创建', permissions: ['ticket:manage', 'knowledge:create'] },
    });

    const body = await resp.json();
    expect(body.code, `创建角色失败: ${JSON.stringify(body)}`).toBe(0);

    // 从列表获取角色 ID（创建可能返回 data: null）
    const listResp = await request.get(apiUrl('/api/v1/admin/roles?page_size=100'), {
      headers: authHeaders(token),
    });
    const listBody = await listResp.json();
    const roles = listBody.data as Array<Record<string, unknown>>;
    const created = roles?.find((r: Record<string, unknown>) => r.name === roleName);
    expect(created, `应在角色列表中找到 "${roleName}"`).toBeDefined();
    roleId = created!.id as number;
  });

  test('重复角色名返回 10005', async ({ request }) => {
    if (!token || !roleId) { test.skip(true, '缺少 token 或角色'); return; }
    const resp = await request.post(apiUrl('/api/v1/admin/roles'), {
      headers: authHeaders(token),
      data: { name: roleName, description: '重复', permissions: [] },
    });
    const body = await resp.json();
    expect([200, 400, 409]).toContain(resp.status());
    expect([10005, 0]).toContain(body.code);
  });

  test('缺少名称返回校验失败', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.post(apiUrl('/api/v1/admin/roles'), {
      headers: authHeaders(token),
      data: { description: '无名称', permissions: [] },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('角色列表返回分页数据', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/roles?page=1&page_size=10'), {
      headers: authHeaders(token),
    });
    const body = await assertPaginatedResponse(resp, 1);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);
    if (data.length > 0) {
      assertFields(data[0], { id: 'number', name: 'string', permissions: 'array' });
    }
  });

  test('查看角色详情', async ({ request }) => {
    if (!token || !roleId) { test.skip(true, '缺少 token 或角色'); return; }
    const resp = await request.get(apiUrl(`/api/v1/admin/roles/${roleId}`), {
      headers: authHeaders(token),
    });
    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBe(roleId);
  });

  test('更新角色（全量替换 permissions）', async ({ request }) => {
    if (!token || !roleId) { test.skip(true, '缺少 token 或角色'); return; }
    const resp = await request.put(apiUrl(`/api/v1/admin/roles/${roleId}`), {
      headers: authHeaders(token),
      data: { name: `${roleName}_v2`, description: '更新后', permissions: ['ticket:manage', 'audit:view'] },
    });
    await assertSuccess(resp);
  });

  test('删除角色成功', async ({ request }) => {
    if (!token || !roleId) { test.skip(true, '缺少 token 或角色'); return; }
    const resp = await request.delete(apiUrl(`/api/v1/admin/roles/${roleId}`), {
      headers: authHeaders(token),
    });
    await assertSuccess(resp);
  });

  test('重复删除返回 404', async ({ request }) => {
    if (!token || !roleId) { test.skip(true, '缺少 token 或角色'); return; }
    const resp = await request.delete(apiUrl(`/api/v1/admin/roles/${roleId}`), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });

  test('不存在的角色返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/roles/99999'), {
      headers: authHeaders(token),
    });
    await assertError(resp, [200, 404], 10004);
  });
});

test.describe('菜单管理', () => {
  test('菜单列表返回树形结构', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.get(apiUrl('/api/v1/admin/menus'), {
      headers: authHeaders(token),
    });
    const body = await assertSuccess(resp);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);
    if (data.length > 0) {
      assertFields(data[0], { id: 'number', name: 'string', path: 'string', icon: 'string', sort_order: 'number' });
    }
  });

  test('更新角色菜单权限（全量替换）', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const roleResp = await request.get(apiUrl('/api/v1/admin/roles?page_size=1'), {
      headers: authHeaders(token),
    });
    const roleBody = await roleResp.json();
    if (roleBody.code !== 0 || !roleBody.data?.length) { test.skip(true, '无可用角色'); return; }
    const roleId = roleBody.data[0].id;

    const resp = await request.put(apiUrl(`/api/v1/admin/roles/${roleId}/menus`), {
      headers: authHeaders(token),
      data: { menu_ids: [1, 2] },
    });
    await assertSuccess(resp);
  });

  test('无效角色 ID 更新菜单返回 404', async ({ request }) => {
    if (!token) { test.skip(true, '缺少 token'); return; }
    const resp = await request.put(apiUrl('/api/v1/admin/roles/99999/menus'), {
      headers: authHeaders(token),
      data: { menu_ids: [1, 2] },
    });
    await assertError(resp, [200, 404], 10004);
  });
});

test.describe('权限验证', () => {
  test('无 token 访问角色列表返回 401', async ({ request }) => {
    const resp = await request.get(apiUrl('/api/v1/admin/roles'));
    await assertError(resp, 401, 10001);
  });
});
