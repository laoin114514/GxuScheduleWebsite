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

// New 组装路由。上传接口鉴权；最新版查询公开但限流。
func New(cfg *config.Config, h *handler.Handler) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSAllowOrigins))

	limiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	api := r.Group("/api/v1")
	api.POST("/releases/upload", middleware.APIKeyAuth(cfg.UploadAPIKey), h.UploadRelease)
	api.GET("/apps/:appKey/latest", middleware.RateLimit(limiter), h.LatestRelease)

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
