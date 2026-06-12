/**
 * 共享类型定义
 *
 * 本目录集中管理项目中多处使用的类型，避免在各 API 模块中重复定义。
 *
 * TODO(types): 创建 api.ts 统一 ApiResponse<T>/PageResponse<T>，消除 api/auth.ts 和
 *            api/dashboard.ts 中的重复定义。
 * TODO(types): 创建 menu.ts 统一 MenuItem 类型，替代 api/auth.ts 中的 any[]。
 * TODO(types): 创建 ticket.ts 统一 TicketItem/TicketDetail/urgencyText 枚举映射，
 *            替代 4 个视图中的重复类型定义。
 * TODO(types): 创建 knowledge.ts 统一知识状态/处理状态枚举，替代多个视图中的重复映射。
 */
