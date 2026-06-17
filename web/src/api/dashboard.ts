/**
 * 看板 API 封装
 *
 * 提供数据看板的统计数据、趋势数据查询。
 *
 */
import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'

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
  // TODO(api/dashboard): 后端当前未实现 granularity，传 week 不会生效。
  // 要么后端支持 day/week，要么前端类型移除此字段。
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
