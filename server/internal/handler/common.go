// Package handler 实现 HTTP 请求处理。
//
// common.go 提供所有 Handler 共享的工具函数。
// 这些函数原本分散在各个 handler 文件中（分页参数解析、ID 解析等），
// 集中到这里以减少重复、统一行为。
package handler

import (
	"strconv"

	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// parsePagination 从查询参数中解析分页参数（page, pageSize）。
//
// 默认值：page=1, pageSize=10。上限：pageSize≤100。
// 为什么集中而非各 handler 自行解析：
// 6 个 handler 原本各自实现相同的 5 行逻辑，集中后分页策略（默认值、上限）
// 只需在一处修改。
func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return page, pageSize
}

// parseID 从路径参数中解析 int64 ID，解析失败时自动返回错误响应。
//
// 返回值 ok=false 表示解析失败，调用方应直接 return。
// 为什么在 parseID 内部处理响应而非让调用方处理：
// 每个调用方都写一样的错误响应是重复路径，此处统一处理保证错误信息一致。
func parseID(c *gin.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的 "+key)
		return 0, false
	}
	return id, true
}

// getCurrentUserID 从 Gin context 中获取当前用户 ID。
//
// JWTAuth 中间件将当前用户 ID 以 int64 类型写入 context，key 为 "userID"。
// 测试环境中可能不存在，返回 0 作为默认值。
// TODO: 返回 0 静默表示「未认证」— 用于 config.Update 等操作时可能记录错误的 updatedBy=0。
// 应添加 exists 返回值，或额外提供 requireCurrentUserID 版本在未认证时拒绝。
func getCurrentUserID(c *gin.Context) int64 {
	if val, exists := c.Get("userID"); exists {
		if id, ok := val.(int64); ok {
			return id
		}
	}
	return 0
}
