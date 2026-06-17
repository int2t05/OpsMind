import { test as setup, expect } from '@playwright/test';
import { saveAuthState, ApiResponse } from '../utils/test-helpers.js';

/**
 * 认证设置 — 登录一次，保存 token 供所有 API 测试复用。
 *
 * 为什么用 setup 项目模式：
 *   90+ API 测试如果每个都登录一次，不仅慢（每次 200ms+），
 *   还可能在速率限制下触发 429。setup 项目登录一次，所有测试
 *   通过 auth-state.json 共享 token，3 次登录覆盖全部角色。
 *
 * 角色覆盖：
 *   - admin（系统管理员）：全部后台权限
 *   - reporter（报障人）：门户端权限
 *   如需运维人员/知识库管理员角色，在对应测试文件中按需登录。
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: { id: number; username: string };
  roles: string[];
}

/**
 * 执行登录并返回认证状态。
 */
async function doLogin(
  request: Parameters<typeof setup>[0]['request'],
  username: string,
  password: string,
): Promise<{ accessToken: string; refreshToken: string; userId: number; roles: string[] }> {
  const resp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
    data: { username, password },
  });
  expect(resp.status()).toBe(200);
  const body: ApiResponse<LoginResponse> = await resp.json();
  expect(body.code).toBe(0);

  return {
    accessToken: body.data!.access_token,
    refreshToken: body.data!.refresh_token,
    userId: body.data!.user.id,
    roles: body.data!.roles,
  };
}

setup('管理员登录并保存认证状态', async ({ request }) => {
  const adminUser = process.env.TEST_ADMIN_USER || 'admin';
  const adminPass = process.env.TEST_ADMIN_PASS || 'Admin@123';

  const result = await doLogin(request, adminUser, adminPass);

  saveAuthState({
    accessToken: result.accessToken,
    refreshToken: result.refreshToken,
    userId: result.userId,
    username: adminUser,
    roles: result.roles,
    // Token 有效期 2 小时
    expiresAt: Date.now() + 2 * 60 * 60 * 1000,
  });

  // 同时保存一份报障人角色的状态（使用报障人账号或 admin 亦可访问门户）
  const reporterUser = process.env.TEST_REPORTER_USER || 'reporter';
  const reporterPass = process.env.TEST_REPORTER_PASS || 'Reporter@123';

  try {
    const reporterResult = await doLogin(request, reporterUser, reporterPass);

    // 保存到单独文件
    const { fileURLToPath } = await import('url');
    const fs = await import('fs');
    const pathModule = await import('path');
    const __filenameAuth = fileURLToPath(import.meta.url);
    const __dirnameAuth = pathModule.dirname(__filenameAuth);
    const dir = __dirnameAuth;
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(
      pathModule.join(dir, 'auth-state-reporter.json'),
      JSON.stringify(
        {
          accessToken: reporterResult.accessToken,
          refreshToken: reporterResult.refreshToken,
          userId: reporterResult.userId,
          username: reporterUser,
          roles: reporterResult.roles,
          expiresAt: Date.now() + 2 * 60 * 60 * 1000,
        },
        null,
        2,
      ),
      'utf-8',
    );
  } catch {
    // 报障人账号可能不存在，仅记录日志不阻塞测试
    console.warn('⚠ 报障人账号不可用，门户端测试将使用管理员 token');
  }
});
