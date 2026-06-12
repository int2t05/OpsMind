// Package response 提供统一的 JSON 响应格式封装。
//
// 所有 API 响应使用统一格式：{"code": 0, "message": "success", "data": {...}}
// 错误响应根据错误码自动映射 HTTP 状态码，映射规则见 mapHTTPStatus 函数。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"opsmind/pkg/errcode"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 返回成功响应，HTTP 状态码 200
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success,
		Message: "success",
		Data:    data,
	})
}

// Error 返回错误响应，根据错误码自动映射 HTTP 状态码
func Error(c *gin.Context, code int, message string) {
	c.JSON(mapHTTPStatus(code), Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// SuccessWithPage 返回分页成功响应
func SuccessWithPage(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, PageResponse{
		Code:     errcode.Success,
		Message:  "success",
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// mapHTTPStatus 将业务错误码映射为 HTTP 状态码
//
// TODO: ErrAIUnavailable(20001)、ErrRAGUnavailable(20002)、ErrStorageUnavailable(20003)
// 应映射为 503 Service Unavailable，当前 fallthrough 到 500 对客户端语义不明确。
func mapHTTPStatus(code int) int {
	switch code {
	case errcode.ErrAuth:
		return http.StatusUnauthorized
	case errcode.ErrForbidden:
		return http.StatusForbidden
	case errcode.ErrParam:
		return http.StatusBadRequest
	case errcode.ErrNotFound:
		return http.StatusNotFound
	case errcode.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
