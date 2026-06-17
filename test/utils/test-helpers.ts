import { APIRequestContext, APIResponse, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

/**
 * OpsMind API 测试共享工具函数。
 *
 * 提供统一的响应校验、认证 token 管理、分页参数构建、测试数据工厂等功能。
 * 核心设计：requireAuth() 消除各测试文件中重复的 loadAuthState→skip 样板代码。
 */

// ---- 类型定义 ----

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  total?: number;
  page?: number;
  page_size?: number;
}

export interface AuthState {
  accessToken: string;
  refreshToken: string;
  userId: number;
  username: string;
  roles: string[];
  expiresAt: number;
}

export interface LoginData {
  access_token: string;
  refresh_token: string;
  user: { id: number; username: string; real_name: string };
  roles: string[];
  permissions: string[];
  menus: Array<{ id: number; name: string; path: string; icon: string }>;
}

export interface PaginatedData<T> {
  items?: T[];
  articles?: T[];
  total: number;
}

// ---- 认证状态管理 ----

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const AUTH_STATE_PATH = path.resolve(__dirname, '..', 'auth', 'auth-state.json');
const AUTH_REPORTER_PATH = path.resolve(__dirname, '..', 'auth', 'auth-state-reporter.json');

export function loadAuthState(): AuthState | null {
  try {
    if (!fs.existsSync(AUTH_STATE_PATH)) return null;
    const raw = fs.readFileSync(AUTH_STATE_PATH, 'utf-8');
    const state: AuthState = JSON.parse(raw);
    if (Date.now() > state.expiresAt - 30 * 60 * 1000) return null;
    return state;
  } catch {
    return null;
  }
}

/** 加载报障人角色的认证状态 */
export function loadReporterState(): AuthState | null {
  try {
    if (!fs.existsSync(AUTH_REPORTER_PATH)) return null;
    const raw = fs.readFileSync(AUTH_REPORTER_PATH, 'utf-8');
    const state: AuthState = JSON.parse(raw);
    if (Date.now() > state.expiresAt - 30 * 60 * 1000) return null;
    return state;
  } catch {
    return null;
  }
}

export function saveAuthState(state: AuthState): void {
  const dir = path.dirname(AUTH_STATE_PATH);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(AUTH_STATE_PATH, JSON.stringify(state, null, 2), 'utf-8');
}

// ---- 认证 Token 获取（消除 skip 样板） ----

/**
 * 获取 admin token，未找到时抛错让测试显式失败。
 * 用于替代 loadAuthState() + test.skip() 的重复模式。
 *
 * 使用方式：
 *   const token = requireAuth();
 *   如果 token 不可用，测试直接失败（而非静默 skip），
 *   在 CI 中更容易发现 setup 问题。
 */
export function requireAuth(): string {
  const state = loadAuthState();
  if (!state || !state.accessToken) {
    throw new Error('认证状态不可用，请先运行 npm run test:auth');
  }
  return state.accessToken;
}

/** 获取报障人 token */
export function requireReporterAuth(): string {
  const state = loadReporterState();
  if (!state || !state.accessToken) {
    throw new Error('报障人认证状态不可用，请先运行 npm run test:auth');
  }
  return state.accessToken;
}

/** 获取 admin token，不可用时返回 null（调用方自行处理降级） */
export function getToken(): string | null {
  const state = loadAuthState();
  return state?.accessToken || null;
}

// ---- 请求辅助 ----

export const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

/** 构建完整 API URL */
export function apiUrl(path: string): string {
  return `${BASE_URL}${path}`;
}

export function authHeaders(token: string): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };
}

export function authHeadersMultipart(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

// ---- 响应校验 ----

export async function assertSuccess(response: APIResponse, expectedCode = 0): Promise<ApiResponse> {
  expect(response.status()).toBe(200);
  const body: ApiResponse = await response.json();
  expect(body.code).toBe(expectedCode);
  if (expectedCode === 0) {
    expect(body.message).toBe('success');
  }
  return body;
}

export async function assertError(
  response: APIResponse,
  expectedHttpStatus: number | number[],
  expectedCode: number,
): Promise<ApiResponse> {
  const statuses = Array.isArray(expectedHttpStatus) ? expectedHttpStatus : [expectedHttpStatus];
  expect(statuses, `HTTP ${response.status()} not in [${statuses}]`).toContain(response.status());
  const body: ApiResponse = await response.json();
  const codes = Array.isArray(expectedCode) ? expectedCode : [expectedCode];
  expect(codes, `期望 code in [${codes}], got ${body.code}`).toContain(body.code);
  expect(body.message).toBeTruthy();
  return body;
}

export async function assertPaginatedResponse(
  response: APIResponse,
  minTotal = 0,
): Promise<ApiResponse> {
  const body = await assertSuccess(response);
  if (body.total !== undefined) {
    expect(typeof body.total).toBe('number');
    expect(body.total).toBeGreaterThanOrEqual(minTotal);
  }
  return body;
}

export function assertDataNotNull<T>(body: ApiResponse<T>): T {
  expect(body.data).not.toBeNull();
  expect(body.data).not.toBeUndefined();
  return body.data!;
}

/**
 * 校验字段存在且类型正确（用于响应结构契约测试）。
 *
 * @param obj   响应对象
 * @param fields 字段名到期望类型的映射（'string' | 'number' | 'boolean' | 'array' | 'object'）
 */
export function assertFields(
  obj: Record<string, unknown>,
  fields: Record<string, 'string' | 'number' | 'boolean' | 'array' | 'object'>,
): void {
  for (const [key, type] of Object.entries(fields)) {
    expect(obj, `字段 ${key} 应存在`).toHaveProperty(key);
    switch (type) {
      case 'array':
        expect(Array.isArray(obj[key]), `字段 ${key} 应为数组`).toBe(true);
        break;
      case 'object':
        expect(typeof obj[key], `字段 ${key} 应为对象`).toBe('object');
        expect(obj[key]).not.toBeNull();
        break;
      default:
        expect(typeof obj[key], `字段 ${key} 应为 ${type}`).toBe(type);
    }
  }
}

// ---- 测试数据工厂 ----

let seq = 0;
export function uniqueName(prefix: string): string {
  seq++;
  return `${prefix}_${Date.now()}_${seq}`;
}

export function uniqueUsername(): string {
  return `testuser_${Date.now()}_${++seq}`;
}

export function validPassword(): string {
  return `Test${Date.now()}!1`;
}

/** 标准测试文章数据 */
export function testArticleData(title?: string) {
  return {
    title: title || uniqueName('测试文章'),
    content: `## 自动化测试\n\n这是测试内容，创建于 ${new Date().toISOString()}。\n\n### 操作步骤\n\n1. 第一步\n2. 第二步\n3. 第三步`,
    source_type: 1,
    category: '测试分类',
    tags: ['测试', '自动化'],
  };
}

/** 标准测试申告数据 */
export function testTicketData(title?: string) {
  return {
    title: title || uniqueName('测试申告'),
    description: '自动化测试创建的申告，用于验证接口功能',
    urgency: 1,
    impact_scope: 0,
    affected_systems: ['测试系统'],
    contact_phone: '13800000001',
    contact_email: 'test@opsmind.local',
  };
}

/** 标准测试用户数据 */
export function testUserData() {
  return {
    username: uniqueUsername(),
    password: validPassword(),
    real_name: 'Playwright测试用户',
    phone: `1380000${String(++seq).padStart(4, '0')}`,
    email: `test_${Date.now()}@opsmind.local`,
    role_ids: [4],
  };
}
