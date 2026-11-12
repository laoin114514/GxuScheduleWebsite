package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/laoin114514/gxuschedule-server/internal/auth"
)

// 上传 / 管理面鉴权头。
const (
	// UploadKeyHeader 每应用上传 key 的请求头。
	UploadKeyHeader = "X-API-Key"
	// AdminKeyHeader 管理面密钥请求头（也接受 Authorization: Bearer）。
	AdminKeyHeader = "X-Admin-Key"
)

// CtxAdminActor 管理面请求在 gin context 里的标记。
const CtxAdminActor = "adminActor"

// AdminAuth 管理面鉴权中间件：常量时间比较多把 admin key（支持轮换），
// 空配置直接 503，绝不允许裸奔。上传 key 没有管理权限。
func AdminAuth(adminKeys []string) gin.HandlerFunc {
	if len(adminKeys) == 0 {
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code": 503, "errorCode": "SERVER_NOT_CONFIGURED", "message": "admin auth not configured",
			})
		}
	}
	return func(c *gin.Context) {
		got := adminKeyFromRequest(c)
		if got == "" || !auth.ConstantTimeMatchAny(adminKeys, got) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "errorCode": "ADMIN_UNAUTHORIZED", "message": "invalid admin key",
			})
			return
		}
		c.Set(CtxAdminActor, "admin")
		c.Next()
	}
}

// IsAdmin 当前请求是否通过了 AdminAuth（仅 admin 路由组内有效）。
func IsAdmin(c *gin.Context) bool {
	return c.GetString(CtxAdminActor) == "admin"
}

// adminKeyFromRequest 从 Authorization: Bearer 或 X-Admin-Key 取管理面 key。
func adminKeyFromRequest(c *gin.Context) string {
	if header := strings.TrimSpace(c.GetHeader("Authorization")); header != "" {
		if len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
			return strings.TrimSpace(header[7:])
		}
		return ""
	}
	return strings.TrimSpace(c.GetHeader(AdminKeyHeader))
}
