// 业务错误码：定义错误码枚举、HTTP 状态码映射、错误包装工具
package errors

import (
	"fmt"
	"net/http"
)

// Code 业务错误码类型
type Code string

// 通用错误码 — 与 docs/API/common/overview.md 第 4 节对齐
const (
	CodeValidationError     Code = "validation_error"
	CodeInvalidJSON         Code = "invalid_json"
	CodeUnauthorized        Code = "unauthorized"
	CodeForbidden           Code = "forbidden"
	CodeNotFound            Code = "not_found"
	CodeConflict            Code = "conflict"
	CodeRateLimitExceeded   Code = "rate_limit_exceeded"
	CodeUpstreamUnavailable Code = "upstream_unavailable"
	CodeServiceUnavailable  Code = "service_unavailable"
	CodeInternalError       Code = "internal_error"

	// 业务特有错误码
	CodeAccountFrozen              Code = "account_frozen"
	CodeInvalidStatusTransition    Code = "invalid_status_transition"
	CodeContactMismatch            Code = "contact_mismatch"
	CodeEmbeddingConfigLocked      Code = "embedding_config_locked"
	CodeInvalidTicketStatus        Code = "invalid_ticket_status"
	CodeFileTooLarge               Code = "file_too_large"
	CodeUnsupportedFileType        Code = "unsupported_file_type"
	CodeDuplicateKnowledgeCandidate Code = "duplicate_knowledge_candidate"
)

// HTTPStatus 返回错误码对应的 HTTP 状态码
func (c Code) HTTPStatus() int {
	switch c {
	case CodeInvalidJSON:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeEmbeddingConfigLocked, CodeDuplicateKnowledgeCandidate:
		return http.StatusConflict
	case CodeValidationError, CodeInvalidStatusTransition, CodeInvalidTicketStatus:
		return http.StatusUnprocessableEntity
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case CodeUpstreamUnavailable:
		return http.StatusBadGateway
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// FieldError 字段级校验错误
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// AppError 业务错误，包含错误码、消息和详情
type AppError struct {
	Code    Code         `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New 创建一个简单的业务错误
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// WithDetails 创建带字段详情的业务错误
func WithDetails(code Code, message string, details []FieldError) *AppError {
	return &AppError{Code: code, Message: message, Details: details}
}

// Wrap 包装底层错误为业务错误，保留原始信息用于日志
func Wrap(code Code, message string, err error) *AppError {
	if err != nil {
		return &AppError{Code: code, Message: fmt.Sprintf("%s: %s", message, err.Error())}
	}
	return &AppError{Code: code, Message: message}
}
