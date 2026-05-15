// 统一响应格式：成功 data/meta，失败 error，与 docs/API/common/overview.md 第 3-4 节对齐
package response

import (
	"net/http"

	"opsmind/server/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Meta 分页元数据
type Meta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int64 `json:"total_pages"`
}

// OK 成功返回单对象或列表，HTTP 200
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Created 创建成功，HTTP 201
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// List 成功返回分页列表
func List(c *gin.Context, data interface{}, meta Meta) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": meta,
	})
}

// NoContent 成功但无响应体，HTTP 204
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error 统一错误响应，根据 AppError.Code 自动设置 HTTP 状态码
func Error(c *gin.Context, err *errors.AppError) {
	status := err.Code.HTTPStatus()
	c.AbortWithStatusJSON(status, gin.H{"error": err})
}

// ErrorRaw 原始错误响应，用于未包装为标准 AppError 的场景
func ErrorRaw(c *gin.Context, status int, code errors.Code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
