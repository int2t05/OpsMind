// Panic 恢复中间件：捕获 panic 并返回统一错误，不导致进程退出
package middleware

import (
	"net/http"

	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/logger"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				response.ErrorRaw(c, http.StatusInternalServerError, errors.CodeInternalError, "服务内部异常")
			}
		}()
		c.Next()
	}
}
