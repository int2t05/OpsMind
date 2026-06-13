/**
 * 系统配置 API 封装（后台管理端）
 *
 * 提供系统配置项的读写接口（键值对形式）。
 */
import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'

/** 配置值响应 */
export interface ConfigValue {
  key: string
  value: string
  description: string
  updated_at: string
}

/** 获取系统配置 */
export function getConfig(key: string) {
  return request.get<ApiResponse<ConfigValue>>(`/api/v1/admin/configs/${key}`)
}

/** 设置系统配置（支持 number 自动转 string） */
export function setConfig(key: string, value: string | number) {
  return request.put<ApiResponse<ConfigValue>>(`/api/v1/admin/configs/${key}`, { value: String(value) })
}
