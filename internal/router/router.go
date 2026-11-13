package router

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/laoin114514/gxuschedule-server/internal/config"
	"github.com/laoin114514/gxuschedule-server/internal/handler"
	"github.com/laoin114514/gxuschedule-server/internal/middleware"
	"github.com/laoin114514/gxuschedule-server/internal/web"
)

// New 组装路由。
//
//   - 公开接口：最新版查询（按 IP 限流）；上传接口的鉴权在 handler 内部完成（便于失败审计），
//     因此这里不再挂上传鉴权中间件，避免出现「中间件先拦截、handler 记不到审计」的问题。
//   - 管理面：/api/v1/admin/*，用 ADMIN_API_KEYS 鉴权 + 独立 IP 限流。
func New(cfg *config.Config, h *handler.Handler) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		middleware.RequestID(),
		middleware.CORS(cfg.CORSAllowOrigins),
	)

	// 两个独立限流器：公开接口与管理面互不挤占配额。
	publicLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	adminLimiter := middleware.NewIPRateLimiter(cfg.AdminRateRPS, cfg.AdminRateBurst)

	api := r.Group("/api/v1")
	api.POST("/releases/upload", h.UploadRelease)
	api.GET("/apps/:appKey/latest", middleware.RateLimit(publicLimiter), h.LatestRelease)

	admin := api.Group("/admin",
		middleware.AdminAuth(cfg.AdminAPIKeys),
		middleware.RateLimit(adminLimiter),
	)
	{
		// apps
		admin.POST("/apps", h.AdminCreateApp)
		admin.GET("/apps", h.AdminListApps)
		admin.GET("/apps/:appKey", h.AdminGetApp)
		admin.PATCH("/apps/:appKey", h.AdminUpdateApp)
		admin.GET("/apps/:appKey/usage", h.AdminAppUsage)
		// upload keys
		admin.POST("/apps/:appKey/keys", h.AdminIssueKey)
		admin.GET("/apps/:appKey/keys", h.AdminListKeys)
		admin.POST("/keys/:keyID/rotate", h.AdminRotateKey)
		admin.POST("/keys/:keyID/revoke", h.AdminRevokeKey)
		// releases
		admin.GET("/apps/:appKey/releases", h.AdminListReleases)
		admin.DELETE("/releases/:id", h.AdminYankRelease)
		admin.POST("/releases/:id/restore", h.AdminRestoreRelease)
		// 审计
		admin.GET("/audit-logs", h.AdminListAuditLogs)
	}

	// 官网静态托管（可选）：前端产物与接口同源，一次 compose 起来就能访问。
	// 是否启用会回显在 /healthz 上，方便一条 curl 判断线上跑的是哪一版、有没有挂官网。
	webEnabled := web.Register(r, cfg.WebDir)
	if webEnabled {
		log.Printf("static web hosting enabled, serving %s at /", cfg.WebDir)
	} else {
		log.Printf("static web hosting disabled (WEB_DIR=%q)，/ 只会返回 404", cfg.WebDir)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "web": webEnabled})
	})

	return r
}
