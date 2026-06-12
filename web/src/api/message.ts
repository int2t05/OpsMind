/**
 * 站内消息 API 封装（门户端）
 *
 * 提供消息列表查询、标记已读、未读计数等接口。
 *
 * TODO(api/message): listMessages 和 getUnreadCount 的响应类型缺少 ApiResponse<T> 包装 —
 *                   当前直接使用 MessageListResponse/UnreadCountResponse，无法匹配后端实际
 *                   { code, message, data: {...} } 的响应结构。
 */
import request from '../utils/request'

// =============================================================================
// 类型定义
// =============================================================================

export interface MessageItem {
  id: number
  user_id: number
  type: string              // ticket_supplement 等
  related_type: string      // ticket
  related_id: number
  title: string
  content: string
  is_read: boolean
  created_at: string
}

export interface MessageListResponse {
  items: MessageItem[]
  total: number
}

export interface UnreadCountResponse {
  count: number
}

// =============================================================================
// API 方法
// =============================================================================

/** 查询当前用户的消息列表 */
export function listMessages(page: number = 1, pageSize: number = 10) {
  return request.get<MessageListResponse>('/api/v1/portal/messages', {
    params: { page, page_size: pageSize }
  })
}

/** 标记消息为已读 */
export function markAsRead(id: number) {
  return request.put(`/api/v1/portal/messages/${id}/read`)
}

/** 获取未读消息数 */
export function getUnreadCount() {
  return request.get<UnreadCountResponse>('/api/v1/portal/messages/unread-count')
}
