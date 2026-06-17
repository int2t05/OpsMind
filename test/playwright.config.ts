import { defineConfig } from '@playwright/test';

/**
 * OpsMind API 集成测试 — Playwright 配置。
 *
 * 使用 APIRequestContext（无浏览器）直接测试 REST API，
 * 比浏览器 E2E 测试更快，适合 CI/CD 流水线中的 API 契约验证。
 *
 * 认证策略：
 *   auth-setup 项目登录一次获取 token，保存到文件中；
 *   api-tests 项目复用该 token，避免每次测试都登录。
 */
export default defineConfig({
  testDir: './api',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 4 : 2,
  reporter: [
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ['list'],
    ['json', { outputFile: 'test-results.json' }],
  ],
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    baseURL: process.env.API_BASE_URL || 'http://localhost:8080',
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
    },
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    // 认证设置：登录并保存 token 到文件
    {
      name: 'auth-setup',
      testDir: './auth',
      testMatch: /auth\.setup\.ts$/,
    },
    // API 测试：使用保存的认证状态
    {
      name: 'api-tests',
      testDir: './api',
      dependencies: ['auth-setup'],
      use: {
        // 从 auth-setup 项目读取 token
        storageState: 'auth/auth-state.json',
      },
    },
  ],
});
