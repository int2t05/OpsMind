/**
 * 系统配置 API 封装（后台管理端）
 */
import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'

export function getConfig(key: string) {
  return request.get<ApiResponse<unknown>>(`/api/v1/admin/configs/${key}`)
}

export function setConfig(key: string, value: unknown) {
  return request.put<ApiResponse<null>>(`/api/v1/admin/configs/${key}`, { value })
}
