package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/laoin114514/gxuschedule-server/internal/auth"
	"github.com/laoin114514/gxuschedule-server/internal/config"
	"github.com/laoin114514/gxuschedule-server/internal/middleware"
	"github.com/laoin114514/gxuschedule-server/internal/model"
	"github.com/laoin114514/gxuschedule-server/internal/oss"
)

// appKeyPattern appKey 合法格式：小写字母/数字开头，2-64 位 [a-z0-9_-]。
var appKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

// fileNamePattern 安装包原始文件名白名单：字母/数字开头，仅 [A-Za-z0-9._-]，扩展名 .apk/.ipa/.aab。
var fileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.(apk|ipa|aab)$`)

// contentTypes 按扩展名设置 OSS Content-Type。
var contentTypes = map[string]string{
	"apk": "application/vnd.android.package-archive",
	"ipa": "application/octet-stream",
	"aab": "application/octet-stream",
}

// maxVersionCode Android versionCode 合法上限。
const maxVersionCode int64 = 2100000000

// objectStorage 上传/下载所需的最小对象存储能力。
// 生产用 *oss.Client 实现；集成测试可注入内存 fake，从而不连真实 OSS。
type objectStorage interface {
	PutObject(key, localPath string, size int64, contentType string) error
	URL(key string) (string, error)
	DeleteObject(key string) error
}

// Handler 上传 / 最新版查询 / 管理面接口集合。
//
// db 与 oss 在纯路由单测里可能是 nil：鉴权失败、参数校验、审计写入路径必须安全，
// 只有确实需要读库/上传的成功路径才允许依赖它们。
type Handler struct {
	db            *gorm.DB
	oss           objectStorage
	cfg           *config.Config
	resolver      auth.Resolver
	uploadLimiter *middleware.IPRateLimiter
}

// New 生产构造：resolver 用 DB 实现，上传限流按 key/IP 令牌桶。
// ossClient 为 nil 时不启用对象存储（仅测试/降级场景）。
func New(db *gorm.DB, ossClient *oss.Client, cfg *config.Config) *Handler {
	var store objectStorage
	if ossClient != nil {
		store = ossClient
	}
	return NewWithStorage(db, store, cfg, auth.NewDBResolver(db))
}

// NewWithDeps 允许测试注入自定义 auth.Resolver（ossClient 同样允许 nil）。
func NewWithDeps(db *gorm.DB, ossClient *oss.Client, cfg *config.Config, resolver auth.Resolver) *Handler {
	var store objectStorage
	if ossClient != nil {
		store = ossClient
	}
	return NewWithStorage(db, store, cfg, resolver)
}

// NewWithStorage 允许注入自定义对象存储实现（真实 MySQL 集成测试可绕开 OSS）。
// store 传 nil 表示未配置对象存储。
func NewWithStorage(db *gorm.DB, store objectStorage, cfg *config.Config, resolver auth.Resolver) *Handler {
	if resolver == nil {
		resolver = auth.NewDBResolver(db)
	}
	return &Handler{
		db:            db,
		oss:           store,
		cfg:           cfg,
		resolver:      resolver,
		uploadLimiter: middleware.NewIPRateLimiter(cfg.UploadRateRPS, cfg.UploadRateBurst),
	}
}

// ---------- 上传接口 ----------

// uploadReleaseReq 上传表单。appKey 在 scoped 模式下可省略（key 已绑定 app）；
// legacy 全局 key 上传时必须显式指定。name 字段仅为兼容旧发布脚本保留，
// 上传接口绝不修改 app 显示名（改由 admin API 负责）。
type uploadReleaseReq struct {
	AppKey      string `form:"appKey"`
	VersionCode int64  `form:"versionCode" binding:"required"`
	VersionName string `form:"versionName" binding:"required"`
	Name        string `form:"name"`
	Changelog   string `form:"changelog"`
	Forced      bool   `form:"forced"`
}

// releaseView 上传接口返回的发布信息（字段与旧契约保持一致）。
type releaseView struct {
	ReleaseID   uint   `json:"releaseId"`
	AppKey      string `json:"appKey"`
	VersionCode int64  `json:"versionCode"`
	VersionName string `json:"versionName"`
	FileKey     string `json:"fileKey"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
}

// uploadActor 鉴权成功后的上传身份。
type uploadActor struct {
	app         *model.App
	key         *model.AppUploadKey
	uploadKeyID uint   // 落库到 release.upload_key_id
	publicKeyID string // 审计用公开 keyID
}

// UploadRelease POST /api/v1/releases/upload
//
// 鉴权放在 handler 内部（而不是中间件），这样鉴权失败也能落审计：
//  1. X-API-Key 必须是 app 专属 key（scoped/dual）或旧全局 key（legacy/dual 兜底）；
//  2. admin key 永远不能当上传 key 用；
//  3. key 绑定的 app 与表单 appKey 必须一致；绝不会自动创建 app。
func (h *Handler) UploadRelease(c *gin.Context) {
	maxBytes := h.cfg.MaxUploadMB<<20 + 1<<20 // 额外 1MB 留给表单其他字段
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	raw := strings.TrimSpace(c.GetHeader(middleware.UploadKeyHeader))
	if raw == "" {
		h.uploadDenied(c, nil, "", http.StatusUnauthorized, codeInvalidAPIKey,
			"缺少上传密钥：请在 X-API-Key 请求头携带 app 专属上传 key", "missing_key")
		return
	}
	// 管理面 key 绝不允许用于上传：避免 admin 密钥扩散到 CI/发布脚本。
	if auth.ConstantTimeMatchAny(h.cfg.AdminAPIKeys, raw) {
		h.uploadDenied(c, nil, "", http.StatusUnauthorized, codeInvalidAPIKey,
			"无效的上传密钥", "admin_key_as_upload_key")
		return
	}

	actor, authed := h.authenticateUpload(c, raw)
	if !authed {
		return
	}

	// 参数校验放在鉴权之后：未鉴权的请求不能借参数错误探测内部行为。
	var req uploadReleaseReq
	if err := c.ShouldBind(&req); err != nil {
		if isBodyTooLarge(err) {
			h.uploadDenied(c, actor, "", http.StatusRequestEntityTooLarge, codeFileTooLarge,
				fmt.Sprintf("文件过大，上限 %dMB", h.cfg.MaxUploadMB), "body_too_large")
			return
		}
		h.uploadDenied(c, actor, "", http.StatusBadRequest, codeInvalidParam,
			"参数错误：versionCode/versionName 必填", "invalid_form")
		return
	}
	if req.AppKey != "" && !appKeyPattern.MatchString(req.AppKey) {
		h.uploadDenied(c, actor, "", http.StatusBadRequest, codeInvalidParam,
			"appKey 非法：小写字母/数字开头，仅限 [a-z0-9_-]，长度 2-64", "invalid_app_key")
		return
	}
	// 防止超长文本直接写库抛 500：versionName 按字符计、changelog 按字节计。
	if len([]rune(req.VersionName)) > 64 {
		h.uploadDenied(c, actor, "", http.StatusBadRequest, codeInvalidParam,
			"versionName 过长（最多 64 字符）", "invalid_version_name")
		return
	}
	if len(req.Changelog) > 60000 {
		h.uploadDenied(c, actor, "", http.StatusBadRequest, codeInvalidParam,
			"changelog 过长（最多 60000 字节）", "changelog_too_long")
		return
	}

	app, okApp := h.resolveUploadApp(c, actor, req.AppKey)
	if !okApp {
		return
	}

	// versionCode 合法性放在确定 app 之后校验（与契约顺序一致：先鉴权/绑定，再校验版本号）。
	if req.VersionCode < 1 || req.VersionCode > maxVersionCode {
		h.uploadDenied(c, actor, app.AppKey, http.StatusBadRequest, codeInvalidParam,
			fmt.Sprintf("versionCode 必须在 1..%d 之间", maxVersionCode), "invalid_version_code")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			h.uploadDenied(c, actor, app.AppKey, http.StatusRequestEntityTooLarge, codeFileTooLarge,
				fmt.Sprintf("文件过大，上限 %dMB", h.cfg.MaxUploadMB), "body_too_large")
			return
		}
		h.uploadDenied(c, actor, app.AppKey, http.StatusBadRequest, codeInvalidParam, "缺少文件字段 file", "missing_file")
		return
	}
	fileName := filepath.Base(fileHeader.Filename)
	if !fileNamePattern.MatchString(strings.ToLower(fileName)) {
		h.uploadDenied(c, actor, app.AppKey, http.StatusBadRequest, codeInvalidParam,
			"文件名非法：仅允许字母数字/._-，扩展名 .apk/.ipa/.aab", "invalid_file_name")
		return
	}
	if len(fileName) > 200 {
		h.uploadDenied(c, actor, app.AppKey, http.StatusBadRequest, codeInvalidParam,
			"文件名过长（最多 200 字节）", "file_name_too_long")
		return
	}
	// 单文件上限：全局 MAX_UPLOAD_MB，app 若配置了更小值则以 app 为准。
	limitMB := h.cfg.MaxUploadMB
	if app.MaxUploadMB > 0 && app.MaxUploadMB < limitMB {
		limitMB = app.MaxUploadMB
	}
	if fileHeader.Size > limitMB<<20 {
		h.uploadDenied(c, actor, app.AppKey, http.StatusRequestEntityTooLarge, codeFileTooLarge,
			fmt.Sprintf("文件过大，上限 %dMB", limitMB), "file_too_large")
		return
	}

	// 先落临时文件并算 sha256：幂等判断、OSS 上传、回滚清理都依赖它。
	tmpPath, sum, size, err := saveUploadTempFile(fileHeader)
	if err != nil {
		failInternal(c, "upload: save temp file", err)
		return
	}
	defer os.Remove(tmpPath)

	// 幂等：同 (appKey, versionCode) 已存在。
	var existing model.Release
	err = h.db.Where("app_key = ? AND version_code = ?", app.AppKey, req.VersionCode).First(&existing).Error
	switch {
	case err == nil:
		if existing.SHA256 != sum {
			h.actorAudit(c, actor, "upload.conflict", app.AppKey, "release",
				strconv.FormatUint(uint64(existing.ID), 10), map[string]any{
					"versionCode":    req.VersionCode,
					"sha256":         sum,
					"existingSha256": existing.SHA256,
				})
			fail(c, http.StatusConflict, codeVersionConflict,
				"versionCode 已存在且文件内容不同；请提升 versionCode 后重试")
			return
		}
		if existing.Status == model.ReleaseStatusYanked {
			// 下架后重传同一份包：复活为 live。OSS 对象/哈希/大小都不变，只更新可变元信息。
			updates := map[string]any{
				"status":        model.ReleaseStatusLive,
				"yanked_at":     nil,
				"yanked_by":     "",
				"yank_reason":   "",
				"version_name":  req.VersionName,
				"changelog":     req.Changelog,
				"forced":        req.Forced,
				"upload_key_id": actor.uploadKeyID,
			}
			if uerr := h.db.Model(&model.Release{}).Where("id = ?", existing.ID).Updates(updates).Error; uerr != nil {
				failInternal(c, "upload: resurrect release", uerr)
				return
			}
			existing.Status = model.ReleaseStatusLive
			existing.YankedAt = nil
			existing.YankedBy = ""
			existing.YankReason = ""
			existing.VersionName = req.VersionName
			existing.Changelog = req.Changelog
			existing.Forced = req.Forced
			existing.UploadKeyID = actor.uploadKeyID
			h.actorAudit(c, actor, "upload.resurrect", app.AppKey, "release",
				strconv.FormatUint(uint64(existing.ID), 10), map[string]any{
					"versionCode": req.VersionCode,
					"sha256":      sum,
					"fileName":    existing.FileName,
				})
			h.finishUploadSuccess(c, actor, existing)
			return
		}
		h.actorAudit(c, actor, "upload.idempotent", app.AppKey, "release",
			strconv.FormatUint(uint64(existing.ID), 10), map[string]any{
				"versionCode": req.VersionCode,
				"sha256":      sum,
			})
		h.finishUploadSuccess(c, actor, existing)
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 新版本，继续走新增流程。
	default:
		failInternal(c, "upload: query existing release", err)
		return
	}

	// 配额：只统计「新增」上传；幂等/复活在上面已返回，不计数。
	var todayCount int64
	if err := h.db.Model(&model.Release{}).
		Where("app_key = ? AND created_at >= ?", app.AppKey, startOfToday()).
		Count(&todayCount).Error; err != nil {
		failInternal(c, "upload: count daily releases", err)
		return
	}
	dailyLimit := app.DailyUploadLimit
	if dailyLimit <= 0 {
		dailyLimit = h.cfg.DefaultDailyUploadLimit
	}
	if dailyLimit > 0 && todayCount >= int64(dailyLimit) {
		h.actorAudit(c, actor, "upload.quota", app.AppKey, "release", "", map[string]any{
			"dailyLimit": dailyLimit,
			"usedToday":  todayCount,
		})
		fail(c, http.StatusTooManyRequests, codeUploadQuotaExceeded,
			fmt.Sprintf("今日上传次数已达上限（%d 次）；幂等重传不计数，如确需发布请联系管理员", dailyLimit))
		return
	}

	// 单调性：默认拒绝低于当前最新的 versionCode，防止误发回滚包。
	var maxVC int64
	if err := h.db.Model(&model.Release{}).Where("app_key = ?", app.AppKey).
		Select("COALESCE(MAX(version_code), 0)").Scan(&maxVC).Error; err != nil {
		failInternal(c, "upload: query max version", err)
		return
	}
	if req.VersionCode < maxVC && !(app.AllowDowngrade || h.cfg.AllowDowngrade) {
		h.actorAudit(c, actor, "upload.downgrade", app.AppKey, "release", "", map[string]any{
			"versionCode":    req.VersionCode,
			"maxVersionCode": maxVC,
		})
		fail(c, http.StatusConflict, codeDowngradeRejected,
			fmt.Sprintf("versionCode=%d 低于当前最新 %d；如需回滚请先由管理员开启该 app 的 allowDowngrade", req.VersionCode, maxVC))
		return
	}

	// 上传 OSS 并落库；半失败时尽力删除刚上传的对象（不删除历史版本对象）。
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	fileKey := fmt.Sprintf("release/%s/%d/%s", app.AppKey, req.VersionCode, fileName)
	if h.oss == nil {
		failInternal(c, "upload: oss client not configured", errors.New("nil oss client"))
		return
	}
	if perr := h.oss.PutObject(fileKey, tmpPath, size, contentTypes[ext]); perr != nil {
		log.Printf("upload: oss put object %s failed: %v", fileKey, perr)
		fail(c, http.StatusInternalServerError, codeOSSError, "OSS 上传失败，请稍后重试")
		return
	}

	release := model.Release{
		AppKey:      app.AppKey,
		VersionCode: req.VersionCode,
		VersionName: req.VersionName,
		FileName:    fileName,
		FileKey:     fileKey,
		SHA256:      sum,
		Size:        size,
		Changelog:   req.Changelog,
		Forced:      req.Forced,
		Status:      model.ReleaseStatusLive,
		UploadKeyID: actor.uploadKeyID,
	}
	if ierr := h.db.Create(&release).Error; ierr != nil {
		if isDuplicateEntry(ierr) {
			// 并发上传同版本：另一请求已落库。这里不删对象（fileKey 可能正被对方引用）。
			h.actorAudit(c, actor, "upload.conflict", app.AppKey, "release", "", map[string]any{
				"versionCode": req.VersionCode,
				"sha256":      sum,
				"reason":      "duplicate_insert",
			})
			fail(c, http.StatusConflict, codeVersionConflict, "versionCode 已被并发上传，请提升 versionCode 后重试")
			return
		}
		log.Printf("upload: insert release failed: %v", ierr)
		h.deleteObjectBestEffort(fileKey)
		failInternal(c, "upload: insert release", ierr)
		return
	}

	h.actorAudit(c, actor, "upload.success", app.AppKey, "release",
		strconv.FormatUint(uint64(release.ID), 10), map[string]any{
			"versionCode": req.VersionCode,
			"sha256":      sum,
			"fileName":    fileName,
		})
	h.finishUploadSuccess(c, actor, release)
}

// authenticateUpload 校验 app 专属上传 key，成功后按 key 限流。
// 失败时已写好响应，返回 false。
func (h *Handler) authenticateUpload(c *gin.Context, raw string) (*uploadActor, bool) {
	app, key, err := h.resolver.ResolveUploadKey(c.Request.Context(), raw)
	if err != nil {
		h.uploadAuthFailed(c, err)
		return nil, false
	}
	return h.newScopedActor(c, app, key)
}

// uploadAuthFailed 把 resolver 的 sentinel error 映射成稳定的 HTTP 错误码。
func (h *Handler) uploadAuthFailed(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrAppDisabled):
		h.uploadDenied(c, nil, "", http.StatusForbidden, codeAppDisabled, "该应用已停用，禁止上传", "app_disabled")
	case errors.Is(err, auth.ErrKeyRevoked):
		h.uploadDenied(c, nil, "", http.StatusUnauthorized, codeInvalidAPIKey,
			"上传密钥已吊销，请联系管理员重新签发", "key_revoked")
	case errors.Is(err, auth.ErrKeyExpired):
		h.uploadDenied(c, nil, "", http.StatusUnauthorized, codeInvalidAPIKey,
			"上传密钥已过期，请联系管理员重新签发", "key_expired")
	case errors.Is(err, auth.ErrKeyNotFound), errors.Is(err, auth.ErrMalformedKey):
		h.uploadDenied(c, nil, "", http.StatusUnauthorized, codeInvalidAPIKey, "无效的上传密钥", "key_not_found")
	default:
		log.Printf("upload: resolve upload key failed: %v", err)
		h.uploadDenied(c, nil, "", http.StatusInternalServerError, codeInternal, "鉴权失败，请稍后重试", "auth_internal_error")
	}
}

// newScopedActor app 专属 key 鉴权成功：按 keyID 限流。
func (h *Handler) newScopedActor(c *gin.Context, app *model.App, key *model.AppUploadKey) (*uploadActor, bool) {
	if app == nil || key == nil {
		failInternal(c, "upload: resolve upload key", errors.New("resolver returned nil app/key"))
		return nil, false
	}
	actor := &uploadActor{
		app:         app,
		key:         key,
		uploadKeyID: key.ID,
		publicKeyID: key.KeyID,
	}
	if !h.uploadLimiter.Allow("key:" + key.KeyID) {
		h.uploadDenied(c, actor, app.AppKey, http.StatusTooManyRequests, codeRateLimited,
			"上传过于频繁，请稍后重试", "rate_limited")
		return nil, false
	}
	return actor, true
}

// resolveUploadApp 校验表单 appKey 与 key 绑定的 app 一致（绝不自动创建 app、绝不改 app 名）。
func (h *Handler) resolveUploadApp(c *gin.Context, actor *uploadActor, formAppKey string) (*model.App, bool) {
	app := actor.app
	if formAppKey != "" && formAppKey != app.AppKey {
		h.uploadDenied(c, actor, app.AppKey, http.StatusForbidden, codeAppKeyMismatch,
			fmt.Sprintf("appKey 与上传密钥绑定的应用不一致（该密钥属于 %s）", app.AppKey), "app_key_mismatch")
		return nil, false
	}
	return app, true
}

// finishUploadSuccess 生成下载地址、更新 key 最近使用时间并返回成功响应。
func (h *Handler) finishUploadSuccess(c *gin.Context, actor *uploadActor, release model.Release) {
	url, err := h.signURL(release.FileKey)
	if err != nil {
		failInternal(c, "upload: sign download url", err)
		return
	}
	if actor.key != nil {
		h.resolver.TouchKeyLastUsed(c.Request.Context(), actor.key)
	}
	ok(c, toView(release, url))
}

// uploadDenied 记录 upload.denied 审计并返回错误响应（actor 可为 nil，表示尚未鉴权）。
func (h *Handler) uploadDenied(c *gin.Context, actor *uploadActor, appKey string, status int, errorCode, message, reason string) {
	h.actorAudit(c, actor, "upload.denied", appKey, "upload", "", map[string]any{"reason": reason})
	fail(c, status, errorCode, message)
}

// actorAudit 按上传身份写审计（未鉴权按 anonymous 记）。
func (h *Handler) actorAudit(c *gin.Context, actor *uploadActor, action, appKey, targetType, targetID string, detail map[string]any) {
	actorType, publicKeyID := "anonymous", ""
	if actor != nil {
		actorType = "app"
		publicKeyID = actor.publicKeyID
	}
	h.recordAudit(c, actorType, publicKeyID, action, appKey, targetType, targetID, auditDetail(detail))
}

// signURL 生成下载地址；oss 未初始化（纯路由单测）时返回空串，不视为错误。
func (h *Handler) signURL(fileKey string) (string, error) {
	if h.oss == nil {
		return "", nil
	}
	return h.oss.URL(fileKey)
}

// deleteObjectBestEffort 尽力删除 OSS 对象（回滚用），失败只记日志。
func (h *Handler) deleteObjectBestEffort(fileKey string) {
	if h.oss == nil {
		return
	}
	if err := h.oss.DeleteObject(fileKey); err != nil {
		log.Printf("upload: rollback oss object %s failed: %v", fileKey, err)
	}
}

// ---------- 最新版本查询接口 ----------

// latestView 最新版查询返回结构（字段与官网契约保持一致，勿随意改名）。
type latestView struct {
	AppKey             string `json:"appKey"`
	CurrentVersionCode int64  `json:"currentVersionCode"`
	HasUpdate          bool   `json:"hasUpdate"`
	LatestVersionCode  int64  `json:"latestVersionCode"`
	LatestVersionName  string `json:"latestVersionName"`
	Changelog          string `json:"changelog"`
	DownloadURL        string `json:"downloadUrl"`
	SHA256             string `json:"sha256"`
	FileSize           int64  `json:"fileSize"`
	Forced             bool   `json:"forced"`
	// PublishedAt 发版时间（RFC3339，UTC）。官网「更新日期」用它，缺失时前端不展示日期。
	PublishedAt string `json:"publishedAt"`
	// FileName 安装包原始文件名，便于前端展示 / 排查。
	FileName string `json:"fileName"`
}

// LatestRelease GET /api/v1/apps/{appKey}/latest?versionCode=xxx
// 公开接口（路由层按 IP 限流）。versionCode 为 App 上报的当前版本号（整数比较）。
func (h *Handler) LatestRelease(c *gin.Context) {
	appKey := c.Param("appKey")
	if !appKeyPattern.MatchString(appKey) {
		fail(c, http.StatusBadRequest, codeInvalidParam, "appKey 非法")
		return
	}
	current, err := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("versionCode", "0")), 10, 64)
	if err != nil || current < 0 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "versionCode 必须为非负整数")
		return
	}
	if h.db == nil {
		failInternal(c, "latest: db not configured", errors.New("nil db"))
		return
	}

	view := latestView{AppKey: appKey, CurrentVersionCode: current}

	var app model.App
	err = h.db.Where("app_key = ?", appKey).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 官网兼容：尚未建 app 时仍返回 200 + 空 latestView，由前端回退内置信息。
		ok(c, view)
		return
	}
	if err != nil {
		failInternal(c, "latest: query app", err)
		return
	}
	if app.Status != model.AppStatusActive {
		fail(c, http.StatusNotFound, codeAppNotFound, "应用不存在或已停用")
		return
	}

	var release model.Release
	err = h.db.Where("app_key = ? AND status = ?", appKey, model.ReleaseStatusLive).
		Order("version_code DESC").First(&release).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ok(c, view) // 还没有任何在架发布
		return
	}
	if err != nil {
		failInternal(c, "latest: query release", err)
		return
	}
	if h.oss == nil {
		failInternal(c, "latest: oss client not configured", errors.New("nil oss client"))
		return
	}
	url, uerr := h.oss.URL(release.FileKey)
	if uerr != nil {
		failInternal(c, "latest: sign download url", uerr)
		return
	}

	view.HasUpdate = current < release.VersionCode
	view.LatestVersionCode = release.VersionCode
	view.LatestVersionName = release.VersionName
	view.Changelog = release.Changelog
	view.DownloadURL = url
	view.SHA256 = release.SHA256
	view.FileSize = release.Size
	view.Forced = release.Forced
	if !release.CreatedAt.IsZero() {
		view.PublishedAt = release.CreatedAt.UTC().Format(time.RFC3339)
	}
	view.FileName = release.FileName
	ok(c, view)
}

// ---------- 工具函数 ----------

// toView 把 Release 转成上传接口响应。
func toView(r model.Release, url string) releaseView {
	return releaseView{
		ReleaseID:   r.ID,
		AppKey:      r.AppKey,
		VersionCode: r.VersionCode,
		VersionName: r.VersionName,
		FileKey:     r.FileKey,
		DownloadURL: url,
		SHA256:      r.SHA256,
		Size:        r.Size,
	}
}

// saveUploadTempFile 把上传文件落到临时文件并计算 sha256，返回临时文件路径、哈希与字节数。
func saveUploadTempFile(fileHeader *multipart.FileHeader) (path, sum string, size int64, err error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", "", 0, fmt.Errorf("open upload file: %w", err)
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "release-*")
	if err != nil {
		return "", "", 0, fmt.Errorf("create temp file: %w", err)
	}
	path = tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(path)
	}
	if size, err = io.Copy(tmp, src); err != nil {
		cleanup()
		return "", "", 0, fmt.Errorf("write temp file: %w", err)
	}
	if sum, err = sha256File(tmp); err != nil {
		cleanup()
		return "", "", 0, fmt.Errorf("hash temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		os.Remove(path)
		return "", "", 0, fmt.Errorf("close temp file: %w", err)
	}
	return path, sum, size, nil
}

// sha256File 计算已打开文件内容的 sha256（从头读取）。
func sha256File(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// isBodyTooLarge 判断错误是否为请求体超过 MaxBytesReader 限制。
func isBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "request body too large")
}

// startOfToday 本地时区的今日 0 点（配额/用量统计用）。
func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// isDuplicateEntry MySQL 唯一键冲突（GORM 未开启 TranslateError，只能看错误文本）。
func isDuplicateEntry(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate entry")
}
