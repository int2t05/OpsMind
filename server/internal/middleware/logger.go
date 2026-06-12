// Package middleware 提供 Gin 中间件。
//
// 本文件实现请求日志中间件，输出结构化 JSON 日志到 stdout。
// 日志字段：method、path、status_code、latency（ms）、client_ip。
package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 返回请求日志中间件（输出到 stdout）。
//
// 薄封装 LoggerWithWriter(nil)，为默认日志输出场景提供简化调用。
func Logger() gin.HandlerFunc {
	return LoggerWithWriter(nil)
}

// LoggerWithWriter 返回请求日志中间件，可指定输出目标。
// writer 为 nil 时输出到 stdout（用于生产环境）。
// writer 非 nil 时输出到指定 writer（用于测试）。
func LoggerWithWriter(writer io.Writer) gin.HandlerFunc {
	if writer == nil {
		writer = os.Stdout
	}

	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)

		// 构建日志字段
		logEntry := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
			"latency":     latency.String(),
			"client_ip":   c.ClientIP(),
		}

		// 输出 JSON 格式日志
		jsonBytes, err := json.Marshal(logEntry)
		if err != nil {
			// 序列化失败时输出错误信息，避免日志静默丢失
			fmt.Fprintf(writer, "{\"error\":\"日志序列化失败: %v\"}\n", err)
			return
		}
		fmt.Fprintln(writer, string(jsonBytes))
	}
}
