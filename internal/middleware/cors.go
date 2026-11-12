package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 放行官网等前端跨域读取公开接口。
//
// allowOrigins 含 "*" 时回显任意来源（最新版查询是只读公开接口，默认如此）；
// 否则只放行白名单内的 Origin，避免任意站点读取。
func CORS(allowOrigins []string) gin.HandlerFunc {
	allowAll := false
	allowed := make(map[string]bool, len(allowOrigins))
	for _, o := range allowOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			switch {
			case allowAll:
				c.Header("Access-Control-Allow-Origin", "*")
			case allowed[origin]:
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}
		// 公开接口只需要 GET/POST；PATCH/DELETE 与 admin 头是给浏览器管理台预留的
		// （admin API 目前主要给 curl/CI 用，预检不通过也不影响 cURL）。
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization, X-Admin-Key, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "600")

		// 预检请求直接返回，不进业务
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
