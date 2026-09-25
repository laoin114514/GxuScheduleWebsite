// Package web 把构建好的前端产物挂到根路径，由 Go 服务直接静态托管。
//
// 与接口同源，所以官网不需要配置 VITE_API_BASE_URL，也不存在跨域问题。
package web

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Register 在 dir 存在时注册 SPA 静态托管，返回是否启用。
//
// 行为：
//   - 命中真实文件（assets/xxx.js、logo.png…）→ 直接返回，带缓存头
//   - 其余 GET → 回退 index.html（前端单页应用）
//   - 未匹配到的 /api/... → 返回 JSON 404，不落到前端页面
//   - dir 为空或不存在 → 不注册，保持「纯接口服务」，兼容只部署 App 接口的旧方式
func Register(r *gin.Engine, dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("web: resolve %q failed: %v", dir, err)
		return false
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		log.Printf("web: dir %q not found, static hosting disabled", abs)
		return false
	}
	indexFile := filepath.Join(abs, "index.html")
	if _, err := os.Stat(indexFile); err != nil {
		log.Printf("web: %q has no index.html, static hosting disabled", abs)
		return false
	}

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// 没匹配上的接口要走 JSON 404，否则前端会把 HTML 当接口响应解析
		if strings.HasPrefix(reqPath, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusMethodNotAllowed)
			return
		}

		// path.Clean 锚定到根，天然消掉 .. ，避免目录穿越
		rel := strings.TrimPrefix(path.Clean("/"+reqPath), "/")
		if rel != "" {
			full := filepath.Join(abs, filepath.FromSlash(rel))
			if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
				c.Header("Cache-Control", cacheControl(rel))
				c.File(full)
				return
			}
		}

		// SPA 回退：入口 HTML 不缓存，发版后刷新即可生效
		c.Header("Cache-Control", "no-cache")
		c.File(indexFile)
	})

	return true
}

// cacheControl：Vite 产物文件名带内容哈希，可长期强缓存；其它静态资源给一小时。
func cacheControl(rel string) string {
	if strings.HasPrefix(rel, "assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "public, max-age=3600"
}
