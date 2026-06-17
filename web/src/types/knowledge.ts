/**
 * 知识库相关类型定义
 *
 * 提供文章状态/处理状态枚举映射。
 */

/** 知识文章状态 */
export enum KnowledgeStatus {
  Draft = 1,
  PendingReview = 2,
  Approved = 3,
  Published = 4,
  Disabled = 5,
  Rejected = 6,
}

/** 文档处理状态（后端返回字符串） */
export const ProcessStatus = {
  Pending: 'pending',
  Parsing: 'parsing',
  Chunking: 'chunking',
  Embedding: 'embedding',
  Completed: 'completed',
  Failed: 'failed',
} as const
export type ProcessStatus = (typeof ProcessStatus)[keyof typeof ProcessStatus]

/** 知识文章状态文本映射 */
export const KNOWLEDGE_STATUS_TEXT: Record<number, string> = {
  [KnowledgeStatus.Draft]: '草稿',
  [KnowledgeStatus.PendingReview]: '待审核',
  [KnowledgeStatus.Approved]: '已通过',
  [KnowledgeStatus.Published]: '已发布',
  [KnowledgeStatus.Disabled]: '已停用',
  [KnowledgeStatus.Rejected]: '已驳回',
}

/** 文档处理状态文本映射 */
export const PROCESS_STATUS_TEXT: Record<string, string> = {
  [ProcessStatus.Pending]: '待处理',
  [ProcessStatus.Parsing]: '解析中',
  [ProcessStatus.Chunking]: '分块中',
  [ProcessStatus.Embedding]: '向量化中',
  [ProcessStatus.Completed]: '已完成',
  [ProcessStatus.Failed]: '失败',
}
