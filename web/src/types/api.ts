/**
 * 共享 API 响应类型定义
 *
 * 统一项目中所有 API 模块的请求/响应类型，消除分散在各 API 文件中的重复定义。
 * 后端所有接口均返回 { code: number, message: string, data: T } 格式，
 * axios 响应拦截器已将 AxiosResponse 的 data 层提取，因此 T 直接对应后端的 data 字段。
 *
 * 使用方式：
 *   import type { ApiResponse, PageResponse } from '@/types/api'
 *   request.get<ApiResponse<UserListData>>('/api/v1/admin/users', { params })
 */

/** 后端统一响应包装 — 所有 API 端点均使用此格式 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/** 分页响应 — 后端 SuccessWithPage 把 items 放在 data，total/page/page_size 与 code/message 同层 */
export interface PageResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}
