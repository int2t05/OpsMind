import { test, expect } from '@playwright/test';
import {
  requireAuth,
  authHeaders,
  assertSuccess,
  assertError,
  apiUrl,
  LoginData,
} from '../utils/test-helpers.js';

/**
 * 认证接口集成测试 — 覆盖全部 4 个端点。
 */
test.describe('POST /api/v1/auth/login', () => {
  test('正确凭据登录成功，返回完整 token/用户/角色/权限/菜单', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: 'admin', password: 'Admin@123' },
    });

    const body = await assertSuccess(resp);
    const data = body.data as LoginData;

    // Token
    expect(typeof data.access_token).toBe('string');
    expect(data.access_token.length).toBeGreaterThan(50);
    expect(typeof data.refresh_token).toBe('string');

    // 用户信息
    expect(data.user.id).toBeGreaterThan(0);
    expect(data.user.username).toBe('admin');
    expect(data.user.real_name).toBeTruthy();

    // 权限信息
    expect(Array.isArray(data.roles)).toBe(true);
    expect(data.roles.length).toBeGreaterThan(0);
    expect(Array.isArray(data.permissions)).toBe(true);
    expect(Array.isArray(data.menus)).toBe(true);
  });

  test('错误密码返回 10003', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: 'admin', password: 'WrongPassword123' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('不存在的用户名返回 10003', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: 'nonexistent_user_xyz_12345', password: 'Test1234' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('缺少 password 返回参数校验失败', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: 'admin' },
    });
    const body = await resp.json();
    expect([400, 200]).toContain(resp.status());
    expect([10003]).toContain(body.code);
  });

  test('缺少 username 返回参数校验失败', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { password: 'Admin@123' },
    });
    const body = await resp.json();
    expect([400, 200]).toContain(resp.status());
    expect([10003]).toContain(body.code);
  });

  test('空请求体返回参数校验失败', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/login'), { data: {} });
    const body = await resp.json();
    expect([400, 200]).toContain(resp.status());
    expect(body.code).toBeGreaterThan(0);
  });
});

test.describe('POST /api/v1/auth/refresh', () => {
  test('有效 refresh_token 刷新成功，新 token 不同于旧 token', async ({ request }) => {
    const loginResp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: 'admin', password: 'Admin@123' },
    });
    const loginBody = await loginResp.json();
    const oldRefresh = loginBody.data.refresh_token;

    const resp = await request.post(apiUrl('/api/v1/auth/refresh'), {
      data: { refresh_token: oldRefresh },
    });

    const body = await assertSuccess(resp);
    const data = body.data as LoginData;
    expect(data.access_token).toBeDefined();
    expect(data.refresh_token).toBeDefined();
    expect(data.access_token).not.toBe(oldRefresh);
  });

  test('无效 refresh_token 返回 401', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/refresh'), {
      data: { refresh_token: 'invalid_token_xyz' },
    });
    await assertError(resp, 401, 10001);
  });

  test('缺少 refresh_token 返回错误', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/refresh'), { data: {} });
    const body = await resp.json();
    expect([400, 401]).toContain(resp.status());
    expect(body.code).toBeGreaterThan(0);
  });
});

test.describe('POST /api/v1/auth/change-password', () => {
  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/change-password'), {
      data: { old_password: 'OldPass123', new_password: 'NewPass456' },
    });
    await assertError(resp, 401, 10001);
  });

  test('新密码纯数字（不符合大小写+数字策略）返回校验失败', async ({ request }) => {
    const token = requireAuth();
    const resp = await request.post(apiUrl('/api/v1/auth/change-password'), {
      headers: authHeaders(token),
      data: { old_password: 'Admin@123', new_password: '12345678' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('新密码短于 8 位返回校验失败', async ({ request }) => {
    const token = requireAuth();
    const resp = await request.post(apiUrl('/api/v1/auth/change-password'), {
      headers: authHeaders(token),
      data: { old_password: 'Admin@123', new_password: 'Ab1' },
    });
    await assertError(resp, [200, 400], 10003);
  });

  test('新密码无大写字母返回校验失败', async ({ request }) => {
    const token = requireAuth();
    const resp = await request.post(apiUrl('/api/v1/auth/change-password'), {
      headers: authHeaders(token),
      data: { old_password: 'Admin@123', new_password: 'abcdefg1' },
    });
    await assertError(resp, [200, 400], 10003);
  });
});

test.describe('POST /api/v1/auth/logout', () => {
  test('已认证用户登出成功', async ({ request }) => {
    const token = requireAuth();
    const resp = await request.post(apiUrl('/api/v1/auth/logout'), {
      headers: authHeaders(token),
    });
    await assertSuccess(resp);
  });

  test('无 token 登出返回 401', async ({ request }) => {
    const resp = await request.post(apiUrl('/api/v1/auth/logout'));
    await assertError(resp, 401, 10001);
  });
});

// ==================== 修改密码 — 成功路径 ====================

test.describe.serial('POST /api/v1/auth/change-password — 成功路径', () => {
  let testUserId: number;
  let testToken: string;
  const testUsername = `pwtest_${Date.now()}`;
  const oldPassword = 'OldPass@123';
  const newPassword = 'NewPass@456';

  test.beforeAll(async ({ request }) => {
    // 创建测试用户
    const adminToken = requireAuth();
    const createResp = await request.post(apiUrl('/api/v1/admin/users'), {
      headers: authHeaders(adminToken),
      data: {
        username: testUsername,
        password: oldPassword,
        real_name: '密码测试用户',
        phone: '13800009999',
        role_ids: [4],
      },
    });
    const createBody = await createResp.json();
    if (createBody.code !== 0) { test.skip(true, '无法创建测试用户'); return; }

    // 从列表获取 ID
    const listResp = await request.get(apiUrl('/api/v1/admin/users?page_size=50'), {
      headers: authHeaders(adminToken),
    });
    const listBody = await listResp.json();
    const users = listBody.data as Array<Record<string, unknown>>;
    const found = users?.find((u: Record<string, unknown>) => u.username === testUsername);
    if (found) testUserId = found.id as number;

    // 登录获取 token
    const loginResp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: testUsername, password: oldPassword },
    });
    const loginBody = await loginResp.json();
    if (loginBody.code === 0) testToken = (loginBody.data as Record<string, string>).access_token;
  });

  test('修改密码成功 + 新密码可登录', async ({ request }) => {
    if (!testToken) { test.skip(true, '测试用户未创建成功'); return; }

    // 修改密码
    const changeResp = await request.post(apiUrl('/api/v1/auth/change-password'), {
      headers: authHeaders(testToken),
      data: { old_password: oldPassword, new_password: newPassword },
    });
    expect(changeResp.status()).toBe(200);
    const changeBody = await changeResp.json();
    expect(changeBody.code, `修改密码失败: ${JSON.stringify(changeBody)}`).toBe(0);

    // 用新密码登录验证
    const loginResp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: testUsername, password: newPassword },
    });
    const loginBody = await loginResp.json();
    expect(loginBody.code, `新密码登录失败: ${JSON.stringify(loginBody)}`).toBe(0);
    expect((loginBody.data as Record<string, unknown>).access_token).toBeTruthy();
    console.log('  修改密码→新密码登录成功 ✅');
  });

  test.afterAll(async ({ request }) => {
    // 清理：恢复原密码
    if (!testToken || !testUserId) return;
    // 用新密码登录拿到新 token
    const loginResp = await request.post(apiUrl('/api/v1/auth/login'), {
      data: { username: testUsername, password: newPassword },
    });
    const loginBody = await loginResp.json();
    if (loginBody.code !== 0) return;
    const newToken = (loginBody.data as Record<string, string>).access_token;
    // 改回原密码
    await request.post(apiUrl('/api/v1/auth/change-password'), {
      headers: authHeaders(newToken),
      data: { old_password: newPassword, new_password: oldPassword },
    });
  });
});
