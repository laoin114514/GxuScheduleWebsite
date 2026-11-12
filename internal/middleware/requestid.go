package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// CtxRequestID 请求 ID 在 gin context 里的 key。
const CtxRequestID = "requestID"

// RequestID 为每个请求生成（或透传合法的）请求 ID，写入响应头与 context，
// 审计日志/错误日志用它串联同一次操作。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(CtxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// RequestIDFromContext 读取当前请求 ID（未启用中间件时为空串）。
func RequestIDFromContext(c *gin.Context) string {
	return c.GetString(CtxRequestID)
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// validRequestID 只透传客户端传入的短 ASCII 令牌，防止日志注入/超长头。
func validRequestID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
