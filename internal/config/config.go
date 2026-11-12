package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// 上传鉴权模式。
const (
	// UploadAuthScoped 只接受 app 专属上传 key（新部署默认）。
	UploadAuthScoped = "scoped"
	// UploadAuthDual 迁移期：先按 app 专属 key 解析，解析不到再回退旧全局 key。
	UploadAuthDual = "dual"
	// UploadAuthLegacy 回滚期：只接受旧全局 key（管理面 admin API 仍可用）。
	UploadAuthLegacy = "legacy"
)

// Config 全部来自 .env(或容器环境变量),启动时校验必填项。
type Config struct {
	Port   string
	AppEnv string

	// AdminAPIKeys 管理面密钥（ADMIN_API_KEYS，逗号分隔；支持多把以便轮换）。
	AdminAPIKeys []string
	// UploadAuthMode 上传鉴权模式：scoped(默认) / dual / legacy。
	UploadAuthMode string
	// LegacyUploadAPIKey 旧版全局上传 key（UPLOAD_API_KEY）。仅 dual/legacy 模式使用。
	LegacyUploadAPIKey string
	// LegacyUploadAppKeys 迁移期旧全局 key 允许上传的 appKey 白名单
	// （UPLOAD_LEGACY_APPKEYS，逗号分隔；为空表示不限制）。
	LegacyUploadAppKeys []string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBCharset  string

	OSSEndpoint        string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSBucket          string
	// 空桶私有模式下用预签名 URL,此为签名有效期秒数
	OSSSignExpireSecs int64

	// CDNBaseURL 非空时下载地址 = CDNBaseURL + "/" + 对象 key;为空时用 OSS 预签名 URL
	CDNBaseURL string

	// MaxUploadMB 全局单文件上传上限（app 可在此基础上再收紧）。
	MaxUploadMB int64
	// DefaultDailyUploadLimit 新建 app 未指定时的每日上传次数上限
	// （按自然日统计，只计新增 release，幂等重传不计数）。
	DefaultDailyUploadLimit int
	// AllowDowngrade 全局降级开关（app 也可单独开启；两者任一为 true 即允许）。
	AllowDowngrade bool

	// RateLimitRPS/RateLimitBurst 公开 latest 接口的每 IP 限流。
	RateLimitRPS   float64
	RateLimitBurst int
	// UploadRateRPS/UploadRateBurst 单把上传 key 的令牌桶限流（默认约 12 次/分钟）。
	UploadRateRPS   float64
	UploadRateBurst int
	// AdminRateRPS/AdminRateBurst 管理面按 IP 限流。
	AdminRateRPS   float64
	AdminRateBurst int

	// CORSAllowOrigins 允许跨域读取公开接口的来源；含 "*" 表示不限制。
	CORSAllowOrigins []string

	// WebDir 前端构建产物目录；非空且存在时由本服务静态托管（同源，无需 CORS）。
	WebDir string
}

func Load() (*Config, error) {
	// .env 不存在时忽略(容器部署直接注入环境变量)
	_ = godotenv.Load()

	cfg := &Config{
		Port:                    env("SERVER_PORT", "8080"),
		AppEnv:                  env("APP_ENV", "development"),
		AdminAPIKeys:            envList("ADMIN_API_KEYS", ""),
		UploadAuthMode:          env("UPLOAD_AUTH_MODE", UploadAuthScoped),
		LegacyUploadAPIKey:      strings.TrimSpace(os.Getenv("UPLOAD_API_KEY")),
		LegacyUploadAppKeys:     envList("UPLOAD_LEGACY_APPKEYS", ""),
		DBHost:                  os.Getenv("DB_HOST"),
		DBPort:                  env("DB_PORT", "3306"),
		DBUser:                  os.Getenv("DB_USER"),
		DBPassword:              os.Getenv("DB_PASSWORD"),
		DBName:                  os.Getenv("DB_NAME"),
		DBCharset:               env("DB_CHARSET", "utf8mb4"),
		OSSEndpoint:             os.Getenv("OSS_ENDPOINT"),
		OSSAccessKeyID:          os.Getenv("OSS_ACCESS_KEY_ID"),
		OSSAccessKeySecret:      os.Getenv("OSS_ACCESS_KEY_SECRET"),
		OSSBucket:               os.Getenv("OSS_BUCKET"),
		OSSSignExpireSecs:       envInt64("OSS_SIGN_EXPIRE_SECONDS", 3600),
		CDNBaseURL:              strings.TrimRight(os.Getenv("CDN_BASE_URL"), "/"),
		MaxUploadMB:             envInt64("MAX_UPLOAD_MB", 512),
		DefaultDailyUploadLimit: envInt("DEFAULT_DAILY_UPLOAD_LIMIT", 20),
		AllowDowngrade:          envBool("ALLOW_DOWNGRADE", false),
		RateLimitRPS:            envFloat("RATE_LIMIT_RPS", 2),
		RateLimitBurst:          envInt("RATE_LIMIT_BURST", 10),
		UploadRateRPS:           envFloat("UPLOAD_RATE_RPS", 0.2),
		UploadRateBurst:         envInt("UPLOAD_RATE_BURST", 3),
		AdminRateRPS:            envFloat("ADMIN_RATE_RPS", 0.1),
		AdminRateBurst:          envInt("ADMIN_RATE_BURST", 5),
		CORSAllowOrigins:        envList("CORS_ALLOW_ORIGINS", "*"),
		WebDir:                  strings.TrimSpace(os.Getenv("WEB_DIR")),
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	c.UploadAuthMode = strings.ToLower(strings.TrimSpace(c.UploadAuthMode))
	if c.UploadAuthMode == "" {
		c.UploadAuthMode = UploadAuthScoped
	}

	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	// 管理面密钥必填：admin API 可以创建 app、签发/吊销 key，缺失时不允许降级成裸奔。
	if len(c.AdminAPIKeys) == 0 {
		add("缺少必填配置 ADMIN_API_KEYS（生成方式: openssl rand -hex 32；多把用逗号分隔以支持轮换）")
	}
	for i, key := range c.AdminAPIKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			add("ADMIN_API_KEYS 第 %d 项为空", i+1)
			continue
		}
		c.AdminAPIKeys[i] = key
		if len(key) < 16 {
			add("ADMIN_API_KEYS 第 %d 把长度过短（至少 16 字符，建议 openssl rand -hex 32）", i+1)
		}
		if c.LegacyUploadAPIKey != "" && key == c.LegacyUploadAPIKey {
			add("ADMIN_API_KEYS 不能与 UPLOAD_API_KEY（legacy 上传 key）相同")
		}
	}

	switch c.UploadAuthMode {
	case UploadAuthScoped:
		// 新部署：只需要 app 专属 key。
	case UploadAuthDual, UploadAuthLegacy:
		if c.LegacyUploadAPIKey == "" {
			add("UPLOAD_AUTH_MODE=%s 时必须配置 UPLOAD_API_KEY（旧版全局上传 key）", c.UploadAuthMode)
		}
	default:
		add("UPLOAD_AUTH_MODE 只能是 scoped/dual/legacy，当前 %q", c.UploadAuthMode)
	}

	must := func(key, val string) {
		if val == "" {
			add("缺少必填配置 %s", key)
		}
	}
	must("DB_HOST", c.DBHost)
	must("DB_USER", c.DBUser)
	must("DB_PASSWORD", c.DBPassword)
	must("DB_NAME", c.DBName)
	must("OSS_ENDPOINT", c.OSSEndpoint)
	must("OSS_ACCESS_KEY_ID", c.OSSAccessKeyID)
	must("OSS_ACCESS_KEY_SECRET", c.OSSAccessKeySecret)
	must("OSS_BUCKET", c.OSSBucket)

	if c.MaxUploadMB <= 0 {
		add("MAX_UPLOAD_MB 必须大于 0")
	}
	if c.DefaultDailyUploadLimit <= 0 {
		add("DEFAULT_DAILY_UPLOAD_LIMIT 必须大于 0")
	}
	if c.RateLimitRPS <= 0 || c.RateLimitBurst <= 0 {
		add("RATE_LIMIT_RPS/RATE_LIMIT_BURST 必须大于 0")
	}
	if c.UploadRateRPS <= 0 || c.UploadRateBurst <= 0 {
		add("UPLOAD_RATE_RPS/UPLOAD_RATE_BURST 必须大于 0")
	}
	if c.AdminRateRPS <= 0 || c.AdminRateBurst <= 0 {
		add("ADMIN_RATE_RPS/ADMIN_RATE_BURST 必须大于 0")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "；"))
	}
	return nil
}

// LegacyUploadAllows 判断 legacy（旧全局）key 是否允许上传指定 appKey。
// 白名单为空表示不限制（迁移期默认，配合 UPLOAD_LEGACY_APPKEYS 收紧）。
func (c *Config) LegacyUploadAllows(appKey string) bool {
	if len(c.LegacyUploadAppKeys) == 0 {
		return true
	}
	for _, allowed := range c.LegacyUploadAppKeys {
		if allowed == appKey {
			return true
		}
	}
	return false
}

// UsesLegacyUploadKey 当前模式是否接受旧全局上传 key。
func (c *Config) UsesLegacyUploadKey() bool {
	return c.UploadAuthMode == UploadAuthDual || c.UploadAuthMode == UploadAuthLegacy
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envList 读取逗号分隔的列表，空项自动剔除。
func envList(key, def string) []string {
	parts := strings.Split(env(key, def), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1"
}
