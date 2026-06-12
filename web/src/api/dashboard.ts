/**
 * 看板 API 封装
 *
 * 提供数据看板的统计数据、趋势数据查询。
 *
 * TODO(api/dashboard): ApiResponse<T> 与 api/auth.ts 中重复定义 — 应使用 src/types/api.ts 共享类型。
 */
import request from '../utils/request'

interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface StatsData {
  today_tickets: number
  pending_tickets: number
  processing_tickets: number
  resolved_tickets: number
  today_chats: number
  avg_confidence: number
  knowledge_count: number
}

export interface TrendParams {
  start_date: string
  end_date: string
  granularity: string // 'day' | 'week'
}

export interface TrendDataPoint {
  date: string
  ticket_count: number
  chat_count: number
}

export interface TrendData {
  data_points: TrendDataPoint[]
}

/** 获取看板统计概览 */
export function getStats() {
  return request.get<ApiResponse<StatsData>>('/api/v1/admin/dashboard/stats')
}

/** 获取趋势数据 */
export function getTrends(params: TrendParams) {
  return request.get<ApiResponse<TrendData>>('/api/v1/admin/dashboard/trends', { params })
}
