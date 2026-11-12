package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/laoin114514/gxuschedule-server/internal/auth"
	"github.com/laoin114514/gxuschedule-server/internal/model"
)

// 管理面接口：鉴权已由 middleware.AdminAuth 完成，这里只做业务与审计。
// 全部是 JSON 接口；写操作成功后落 audit_logs，读接口不写审计。
// db 为 nil（纯路由单测）时统一返回 500 INTERNAL，不 panic。

// 事务内使用的哨兵错误。
var (
	errActiveKeyExists = errors.New("app already has an active upload key")
	errKeyNotActive    = errors.New("upload key is not active")
)

// ---------- 响应视图 ----------

type appView struct {
	AppKey           string    `json:"appKey"`
	Name             string    `json:"name"`
	Platform         string    `json:"platform"`
	Status           int8      `json:"status"`
	MaxUploadMB      int64     `json:"maxUploadMB"`
	DailyUploadLimit int       `json:"dailyUploadLimit"`
	AllowDowngrade   bool      `json:"allowDowngrade"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type uploadKeyView struct {
	KeyID        string     `json:"keyId"`
	Last4        string     `json:"last4"`
	Label        string     `json:"label"`
	Status       int8       `json:"status"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	RevokedAt    *time.Time `json:"revokedAt"`
	RevokeReason string     `json:"revokeReason"`
}

// uploadKeyIssue 新建/轮换 key 的响应：plaintext 是明文，只返回这一次；
// 其余字段与 uploadKeyView 一致（不含 app 内部 id，keyId 已是公开标识）。
type uploadKeyIssue struct {
	KeyID     string     `json:"keyId"`
	Plaintext string     `json:"plaintext"`
	Last4     string     `json:"last4"`
	Label     string     `json:"label"`
	Status    int8       `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type createAppResp struct {
	App       appView         `json:"app"`
	UploadKey *uploadKeyIssue `json:"uploadKey,omitempty"`
}

type releaseAdminView struct {
	ReleaseID   uint       `json:"releaseId"`
	AppKey      string     `json:"appKey"`
	VersionCode int64      `json:"versionCode"`
	VersionName string     `json:"versionName"`
	FileName    string     `json:"fileName"`
	FileKey     string     `json:"fileKey"`
	DownloadURL string     `json:"downloadUrl"`
	SHA256      string     `json:"sha256"`
	Size        int64      `json:"size"`
	Changelog   string     `json:"changelog"`
	Forced      bool       `json:"forced"`
	Status      int8       `json:"status"`
	UploadKeyID uint       `json:"uploadKeyId"`
	YankedAt    *time.Time `json:"yankedAt"`
	YankedBy    string     `json:"yankedBy"`
	YankReason  string     `json:"yankReason"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type auditLogView struct {
	ID         uint      `json:"id"`
	RequestID  string    `json:"requestId"`
	ActorType  string    `json:"actorType"`
	ActorKeyID string    `json:"actorKeyId"`
	Action     string    `json:"action"`
	AppKey     string    `json:"appKey"`
	TargetType string    `json:"targetType"`
	TargetID   string    `json:"targetId"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"createdAt"`
}

type usageView struct {
	AppKey            string     `json:"appKey"`
	UploadsToday      int64      `json:"uploadsToday"`
	ReleaseCount      int64      `json:"releaseCount"`
	LiveReleaseCount  int64      `json:"liveReleaseCount"`
	KeyCount          int64      `json:"keyCount"`
	ActiveKeyCount    int64      `json:"activeKeyCount"`
	LatestVersionCode int64      `json:"latestVersionCode"`
	LatestVersionName string     `json:"latestVersionName"`
	LastUploadAt      *time.Time `json:"lastUploadAt"`
}

func toAppView(app model.App) appView {
	return appView{
		AppKey:           app.AppKey,
		Name:             app.Name,
		Platform:         app.Platform,
		Status:           app.Status,
		MaxUploadMB:      app.MaxUploadMB,
		DailyUploadLimit: app.DailyUploadLimit,
		AllowDowngrade:   app.AllowDowngrade,
		CreatedAt:        app.CreatedAt,
		UpdatedAt:        app.UpdatedAt,
	}
}

func toUploadKeyView(key model.AppUploadKey) uploadKeyView {
	return uploadKeyView{
		KeyID:        key.KeyID,
		Last4:        key.Last4,
		Label:        key.Label,
		Status:       key.Status,
		ExpiresAt:    key.ExpiresAt,
		LastUsedAt:   key.LastUsedAt,
		CreatedAt:    key.CreatedAt,
		RevokedAt:    key.RevokedAt,
		RevokeReason: key.RevokeReason,
	}
}

func newUploadKeyIssue(plaintext string, key model.AppUploadKey) *uploadKeyIssue {
	return &uploadKeyIssue{
		KeyID:     key.KeyID,
		Plaintext: plaintext,
		Last4:     key.Last4,
		Label:     key.Label,
		Status:    key.Status,
		ExpiresAt: key.ExpiresAt,
		CreatedAt: key.CreatedAt,
	}
}

// toReleaseAdminView 附带下载地址；oss 未配置（单测）时留空。
func (h *Handler) toReleaseAdminView(release model.Release) releaseAdminView {
	downloadURL := ""
	if h.oss != nil {
		url, err := h.oss.URL(release.FileKey)
		if err != nil {
			// 管理面列表不应因签名失败整体报错，留空并记日志。
			log.Printf("admin: sign url for release %d failed: %v", release.ID, err)
		} else {
			downloadURL = url
		}
	}
	return releaseAdminView{
		ReleaseID:   release.ID,
		AppKey:      release.AppKey,
		VersionCode: release.VersionCode,
		VersionName: release.VersionName,
		FileName:    release.FileName,
		FileKey:     release.FileKey,
		DownloadURL: downloadURL,
		SHA256:      release.SHA256,
		Size:        release.Size,
		Changelog:   release.Changelog,
		Forced:      release.Forced,
		Status:      release.Status,
		UploadKeyID: release.UploadKeyID,
		YankedAt:    release.YankedAt,
		YankedBy:    release.YankedBy,
		YankReason:  release.YankReason,
		CreatedAt:   release.CreatedAt,
	}
}

func toAuditLogView(entry model.AuditLog) auditLogView {
	return auditLogView{
		ID:         entry.ID,
		RequestID:  entry.RequestID,
		ActorType:  entry.ActorType,
		ActorKeyID: entry.ActorKeyID,
		Action:     entry.Action,
		AppKey:     entry.AppKey,
		TargetType: entry.TargetType,
		TargetID:   entry.TargetID,
		IP:         entry.IP,
		UserAgent:  entry.UserAgent,
		Detail:     entry.Detail,
		CreatedAt:  entry.CreatedAt,
	}
}

// ---------- 通用小工具 ----------

// requireDB 管理面所有接口都依赖数据库；db 为 nil 时统一 500，不 panic。
func (h *Handler) requireDB(c *gin.Context) bool {
	if h.db == nil {
		failInternal(c, "admin: db not configured", errors.New("nil db"))
		return false
	}
	return true
}

// adminApp 解析 :appKey 并查 app：非法 400、不存在 404、DB 错误 500。
func (h *Handler) adminApp(c *gin.Context) (*model.App, bool) {
	if !h.requireDB(c) {
		return nil, false
	}
	appKey := strings.TrimSpace(c.Param("appKey"))
	if !appKeyPattern.MatchString(appKey) {
		fail(c, http.StatusBadRequest, codeInvalidParam, "appKey 非法：小写字母/数字开头，仅限 [a-z0-9_-]，长度 2-64")
		return nil, false
	}
	var app model.App
	if err := h.db.Where("app_key = ?", appKey).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, codeAppNotFound, fmt.Sprintf("应用 %s 不存在", appKey))
			return nil, false
		}
		failInternal(c, "admin: query app", err)
		return nil, false
	}
	return &app, true
}

// parseUintParam 解析路径里的正整数 ID。
func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		fail(c, http.StatusBadRequest, codeInvalidParam, fmt.Sprintf("%s 必须为正整数", name))
		return 0, false
	}
	return uint(id), true
}

// bindJSON 解析必填 JSON body。
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		if isBodyTooLarge(err) {
			fail(c, http.StatusRequestEntityTooLarge, codeFileTooLarge, "请求体过大")
			return false
		}
		fail(c, http.StatusBadRequest, codeInvalidParam, "请求体格式错误：需要合法 JSON")
		return false
	}
	return true
}

// bindOptionalJSON 解析可选 JSON body：空 body 视为没有字段（DELETE/PATCH 常见）。
func bindOptionalJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		if isBodyTooLarge(err) {
			fail(c, http.StatusRequestEntityTooLarge, codeFileTooLarge, "请求体过大")
			return false
		}
		fail(c, http.StatusBadRequest, codeInvalidParam, "请求体格式错误：需要合法 JSON")
		return false
	}
	return true
}

// defaultDailyUploadLimit 新建 app 未指定每日上限时的默认值。
func (h *Handler) defaultDailyUploadLimit() int {
	if h.cfg.DefaultDailyUploadLimit > 0 {
		return h.cfg.DefaultDailyUploadLimit
	}
	return 20
}

// ---------- apps ----------

// AdminCreateApp POST /api/v1/admin/apps
type createAppReq struct {
	AppKey           string `json:"appKey"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	MaxUploadMB      *int64 `json:"maxUploadMB"`
	DailyUploadLimit *int   `json:"dailyUploadLimit"`
	AllowDowngrade   *bool  `json:"allowDowngrade"`
	IssueKey         *bool  `json:"issueKey"`
	Label            string `json:"label"`
}

func (h *Handler) AdminCreateApp(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	var req createAppReq
	if !bindJSON(c, &req) {
		return
	}

	appKey := strings.TrimSpace(req.AppKey)
	name := strings.TrimSpace(req.Name)
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = "android"
	}
	if !appKeyPattern.MatchString(appKey) {
		fail(c, http.StatusBadRequest, codeInvalidParam, "appKey 非法：小写字母/数字开头，仅限 [a-z0-9_-]，长度 2-64")
		return
	}
	if name == "" {
		fail(c, http.StatusBadRequest, codeInvalidParam, "name 不能为空")
		return
	}
	if len([]rune(name)) > 128 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "name 过长（最多 128 字符）")
		return
	}
	if len(platform) > 32 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "platform 过长（最多 32 字符）")
		return
	}
	maxUploadMB := int64(0)
	if req.MaxUploadMB != nil {
		maxUploadMB = *req.MaxUploadMB
		if maxUploadMB < 0 || maxUploadMB > h.cfg.MaxUploadMB {
			fail(c, http.StatusBadRequest, codeInvalidParam,
				fmt.Sprintf("maxUploadMB 必须在 0..%d 之间（0 表示沿用全局上限）", h.cfg.MaxUploadMB))
			return
		}
	}
	dailyLimit := h.defaultDailyUploadLimit()
	if req.DailyUploadLimit != nil {
		dailyLimit = *req.DailyUploadLimit
		if dailyLimit <= 0 || dailyLimit > 100000 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "dailyUploadLimit 必须在 1..100000 之间")
			return
		}
	}
	allowDowngrade := false
	if req.AllowDowngrade != nil {
		allowDowngrade = *req.AllowDowngrade
	}
	issueKey := true
	if req.IssueKey != nil {
		issueKey = *req.IssueKey
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "default"
	}
	if len([]rune(label)) > 64 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "label 过长（最多 64 字符）")
		return
	}

	app := model.App{
		AppKey:           appKey,
		Name:             name,
		Platform:         platform,
		Status:           model.AppStatusActive,
		MaxUploadMB:      maxUploadMB,
		DailyUploadLimit: dailyLimit,
		AllowDowngrade:   allowDowngrade,
	}
	var issued *uploadKeyIssue
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&app).Error; err != nil {
			return err
		}
		if !issueKey {
			return nil
		}
		plaintext, keyID, keyHash, last4, gerr := auth.Generate()
		if gerr != nil {
			return gerr
		}
		key := model.AppUploadKey{
			AppID:   app.ID,
			KeyID:   keyID,
			KeyHash: keyHash,
			Last4:   last4,
			Label:   label,
			Status:  model.UploadKeyStatusActive,
		}
		if err := tx.Create(&key).Error; err != nil {
			return err
		}
		issued = newUploadKeyIssue(plaintext, key)
		return nil
	})
	if err != nil {
		if isDuplicateEntry(err) {
			fail(c, http.StatusConflict, codeAppExists,
				fmt.Sprintf("appKey=%s 已存在（appKey 永久保留，停用后也不可复用）", appKey))
			return
		}
		failInternal(c, "admin: create app", err)
		return
	}

	detail := map[string]any{"platform": app.Platform, "issuedKey": issueKey}
	if issued != nil {
		detail["keyId"] = issued.KeyID
	}
	h.recordAudit(c, "admin", "", "app.create", app.AppKey, "app",
		strconv.FormatUint(uint64(app.ID), 10), auditDetail(detail))
	ok(c, createAppResp{App: toAppView(app), UploadKey: issued})
}

// AdminListApps GET /api/v1/admin/apps
func (h *Handler) AdminListApps(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	var apps []model.App
	q := h.db.Model(&model.App{})
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("app_key LIKE ? OR name LIKE ?", like, like)
	}
	if err := q.Order("app_key asc").Find(&apps).Error; err != nil {
		failInternal(c, "admin: list apps", err)
		return
	}
	items := make([]appView, 0, len(apps))
	for _, app := range apps {
		items = append(items, toAppView(app))
	}
	ok(c, gin.H{"items": items})
}

// AdminGetApp GET /api/v1/admin/apps/:appKey
func (h *Handler) AdminGetApp(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	var keys []model.AppUploadKey
	if err := h.db.Where("app_id = ?", app.ID).Order("id desc").Find(&keys).Error; err != nil {
		failInternal(c, "admin: list app keys", err)
		return
	}
	items := make([]uploadKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, toUploadKeyView(key))
	}
	ok(c, gin.H{"app": toAppView(*app), "keys": items})
}

// AdminUpdateApp PATCH /api/v1/admin/apps/:appKey
type updateAppReq struct {
	Name             *string `json:"name"`
	Platform         *string `json:"platform"`
	Status           *int8   `json:"status"`
	MaxUploadMB      *int64  `json:"maxUploadMB"`
	DailyUploadLimit *int    `json:"dailyUploadLimit"`
	AllowDowngrade   *bool   `json:"allowDowngrade"`
}

func (h *Handler) AdminUpdateApp(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	var req updateAppReq
	if !bindOptionalJSON(c, &req) {
		return
	}

	updates := make(map[string]any, 6)
	fields := make([]string, 0, 6)
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			fail(c, http.StatusBadRequest, codeInvalidParam, "name 不能为空")
			return
		}
		if len([]rune(name)) > 128 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "name 过长（最多 128 字符）")
			return
		}
		updates["name"] = name
		fields = append(fields, "name")
	}
	if req.Platform != nil {
		platform := strings.TrimSpace(*req.Platform)
		if platform == "" || len(platform) > 32 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "platform 非法（1..32 字符）")
			return
		}
		updates["platform"] = platform
		fields = append(fields, "platform")
	}
	if req.Status != nil {
		if *req.Status != model.AppStatusActive && *req.Status != model.AppStatusDisabled {
			fail(c, http.StatusBadRequest, codeInvalidParam, "status 只能是 1(active) 或 2(disabled)")
			return
		}
		updates["status"] = *req.Status
		fields = append(fields, "status")
	}
	if req.MaxUploadMB != nil {
		if *req.MaxUploadMB < 0 || *req.MaxUploadMB > h.cfg.MaxUploadMB {
			fail(c, http.StatusBadRequest, codeInvalidParam,
				fmt.Sprintf("maxUploadMB 必须在 0..%d 之间（0 表示沿用全局上限）", h.cfg.MaxUploadMB))
			return
		}
		updates["max_upload_mb"] = *req.MaxUploadMB
		fields = append(fields, "maxUploadMB")
	}
	if req.DailyUploadLimit != nil {
		if *req.DailyUploadLimit <= 0 || *req.DailyUploadLimit > 100000 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "dailyUploadLimit 必须在 1..100000 之间")
			return
		}
		updates["daily_upload_limit"] = *req.DailyUploadLimit
		fields = append(fields, "dailyUploadLimit")
	}
	if req.AllowDowngrade != nil {
		updates["allow_downgrade"] = *req.AllowDowngrade
		fields = append(fields, "allowDowngrade")
	}

	if len(updates) == 0 {
		ok(c, toAppView(*app))
		return
	}
	if err := h.db.Model(&model.App{}).Where("id = ?", app.ID).Updates(updates).Error; err != nil {
		failInternal(c, "admin: update app", err)
		return
	}
	var fresh model.App
	if err := h.db.First(&fresh, app.ID).Error; err != nil {
		failInternal(c, "admin: reload app", err)
		return
	}
	h.recordAudit(c, "admin", "", "app.update", app.AppKey, "app",
		strconv.FormatUint(uint64(app.ID), 10), auditDetail(map[string]any{"fields": fields}))
	ok(c, toAppView(fresh))
}

// ---------- upload keys ----------

// AdminIssueKey POST /api/v1/admin/apps/:appKey/keys
type issueKeyReq struct {
	Label     string `json:"label"`
	ExpiresAt string `json:"expiresAt"`
}

func (h *Handler) AdminIssueKey(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	var req issueKeyReq
	if !bindOptionalJSON(c, &req) {
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "default"
	}
	if len([]rune(label)) > 64 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "label 过长（最多 64 字符）")
		return
	}
	var expiresAt *time.Time
	if raw := strings.TrimSpace(req.ExpiresAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			fail(c, http.StatusBadRequest, codeInvalidParam, "expiresAt 必须是 RFC3339 时间（如 2026-01-02T15:04:05Z）")
			return
		}
		expiresAt = &parsed
	}

	var issue *uploadKeyIssue
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 锁 app 行，保证「每 app 同时仅 1 把 active key」不变量。
		var locked model.App
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", app.ID).First(&locked).Error; err != nil {
			return err
		}
		var active model.AppUploadKey
		err := tx.Where("app_id = ? AND status = ?", app.ID, model.UploadKeyStatusActive).First(&active).Error
		if err == nil {
			return errActiveKeyExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		plaintext, keyID, keyHash, last4, gerr := auth.Generate()
		if gerr != nil {
			return gerr
		}
		key := model.AppUploadKey{
			AppID:     app.ID,
			KeyID:     keyID,
			KeyHash:   keyHash,
			Last4:     last4,
			Label:     label,
			Status:    model.UploadKeyStatusActive,
			ExpiresAt: expiresAt,
		}
		if err := tx.Create(&key).Error; err != nil {
			return err
		}
		issue = newUploadKeyIssue(plaintext, key)
		return nil
	})
	if err != nil {
		if errors.Is(err, errActiveKeyExists) {
			fail(c, http.StatusConflict, codeActiveKeyExists,
				"该应用已有 active key；请调用 rotate 接口轮换（旧 key 进入宽限期后自动失效）")
			return
		}
		if isDuplicateEntry(err) {
			fail(c, http.StatusConflict, codeActiveKeyExists, "并发签发冲突，请重试")
			return
		}
		failInternal(c, "admin: issue upload key", err)
		return
	}

	h.recordAudit(c, "admin", "", "key.issue", app.AppKey, "uploadKey", issue.KeyID,
		auditDetail(map[string]any{"label": label, "expiresAt": issue.ExpiresAt}))
	ok(c, issue)
}

// AdminListKeys GET /api/v1/admin/apps/:appKey/keys
func (h *Handler) AdminListKeys(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	var keys []model.AppUploadKey
	if err := h.db.Where("app_id = ?", app.ID).Order("id desc").Find(&keys).Error; err != nil {
		failInternal(c, "admin: list app keys", err)
		return
	}
	items := make([]uploadKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, toUploadKeyView(key))
	}
	ok(c, gin.H{"items": items})
}

// AdminRotateKey POST /api/v1/admin/keys/:keyID/rotate
type rotateKeyReq struct {
	GraceHours *int   `json:"graceHours"`
	Label      string `json:"label"`
}

func (h *Handler) AdminRotateKey(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	keyID := strings.TrimSpace(c.Param("keyID"))
	if keyID == "" {
		fail(c, http.StatusBadRequest, codeInvalidParam, "keyID 非法")
		return
	}
	var req rotateKeyReq
	if !bindOptionalJSON(c, &req) {
		return
	}
	graceHours := 24
	if req.GraceHours != nil {
		graceHours = *req.GraceHours
	}
	if graceHours < 1 || graceHours > 720 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "graceHours 必须在 1..720 之间")
		return
	}
	if len([]rune(strings.TrimSpace(req.Label))) > 64 {
		fail(c, http.StatusBadRequest, codeInvalidParam, "label 过长（最多 64 字符）")
		return
	}

	var (
		oldKey model.AppUploadKey
		newKey model.AppUploadKey
		app    model.App
		issue  *uploadKeyIssue
	)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("key_id = ?", keyID).First(&oldKey).Error; err != nil {
			return err
		}
		// 锁 app 行，与其他发 key/轮换操作串行化。
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", oldKey.AppID).First(&app).Error; err != nil {
			return err
		}
		// 拿到 app 行锁后重新读 key，避免并发轮换基于旧快照重复把同一把 key 当 active。
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("key_id = ?", keyID).First(&oldKey).Error; err != nil {
			return err
		}
		if oldKey.Status != model.UploadKeyStatusActive {
			return errKeyNotActive
		}
		label := strings.TrimSpace(req.Label)
		if label == "" {
			label = oldKey.Label
		}
		now := time.Now()
		deadline := now.Add(time.Duration(graceHours) * time.Hour)
		// 宽限期只延长不缩短：已有更早的过期时间则保留较早者。
		retireExpires := oldKey.ExpiresAt
		if retireExpires == nil || retireExpires.After(deadline) {
			d := deadline
			retireExpires = &d
		}
		if err := tx.Model(&model.AppUploadKey{}).Where("id = ?", oldKey.ID).
			Updates(map[string]any{
				"status":     model.UploadKeyStatusRetiring,
				"expires_at": retireExpires,
			}).Error; err != nil {
			return err
		}
		oldKey.Status = model.UploadKeyStatusRetiring
		oldKey.ExpiresAt = retireExpires

		plaintext, newKeyID, keyHash, last4, gerr := auth.Generate()
		if gerr != nil {
			return gerr
		}
		newKey = model.AppUploadKey{
			AppID:   oldKey.AppID,
			KeyID:   newKeyID,
			KeyHash: keyHash,
			Last4:   last4,
			Label:   label,
			Status:  model.UploadKeyStatusActive,
		}
		if err := tx.Create(&newKey).Error; err != nil {
			return err
		}
		issue = newUploadKeyIssue(plaintext, newKey)
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			fail(c, http.StatusNotFound, codeKeyNotFound, fmt.Sprintf("上传 key %s 不存在", keyID))
		case errors.Is(err, errKeyNotActive):
			fail(c, http.StatusBadRequest, codeKeyNotActive, "仅 active 状态的上传 key 可以轮换")
		default:
			failInternal(c, "admin: rotate upload key", err)
		}
		return
	}

	h.recordAudit(c, "admin", "", "key.rotate", app.AppKey, "uploadKey", newKey.KeyID,
		auditDetail(map[string]any{"oldKeyId": keyID, "graceHours": graceHours, "label": newKey.Label}))
	ok(c, gin.H{"uploadKey": issue, "retireKey": toUploadKeyView(oldKey)})
}

// AdminRevokeKey POST /api/v1/admin/keys/:keyID/revoke
type revokeKeyReq struct {
	Reason string `json:"reason"`
}

func (h *Handler) AdminRevokeKey(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	keyID := strings.TrimSpace(c.Param("keyID"))
	if keyID == "" {
		fail(c, http.StatusBadRequest, codeInvalidParam, "keyID 非法")
		return
	}
	var req revokeKeyReq
	if !bindOptionalJSON(c, &req) {
		return
	}
	reason := truncate(strings.TrimSpace(req.Reason), 255)

	var key model.AppUploadKey
	if err := h.db.Where("key_id = ?", keyID).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, codeKeyNotFound, fmt.Sprintf("上传 key %s 不存在", keyID))
			return
		}
		failInternal(c, "admin: query upload key", err)
		return
	}
	if key.Status == model.UploadKeyStatusRevoked {
		ok(c, toUploadKeyView(key)) // 幂等
		return
	}

	now := time.Now()
	if err := h.db.Model(&model.AppUploadKey{}).Where("id = ?", key.ID).
		Updates(map[string]any{
			"status":        model.UploadKeyStatusRevoked,
			"revoked_at":    now,
			"revoke_reason": reason,
		}).Error; err != nil {
		failInternal(c, "admin: revoke upload key", err)
		return
	}
	key.Status = model.UploadKeyStatusRevoked
	key.RevokedAt = &now
	key.RevokeReason = reason

	appKey := ""
	var app model.App
	if err := h.db.First(&app, key.AppID).Error; err == nil {
		appKey = app.AppKey
	}
	h.recordAudit(c, "admin", "", "key.revoke", appKey, "uploadKey", key.KeyID,
		auditDetail(map[string]any{"reason": reason}))
	ok(c, toUploadKeyView(key))
}

// ---------- releases ----------

// AdminListReleases GET /api/v1/admin/apps/:appKey/releases?includeYanked=1&limit=100
func (h *Handler) AdminListReleases(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "limit 必须在 1..500 之间")
			return
		}
		limit = parsed
	}
	includeYanked := parseBoolLoose(c.Query("includeYanked"))

	q := h.db.Where("app_key = ?", app.AppKey)
	if !includeYanked {
		q = q.Where("status = ?", model.ReleaseStatusLive)
	}
	var releases []model.Release
	if err := q.Order("version_code desc").Limit(limit).Find(&releases).Error; err != nil {
		failInternal(c, "admin: list releases", err)
		return
	}
	items := make([]releaseAdminView, 0, len(releases))
	for _, release := range releases {
		items = append(items, h.toReleaseAdminView(release))
	}
	ok(c, gin.H{"items": items})
}

// AdminYankRelease DELETE /api/v1/admin/releases/:id
type yankReleaseReq struct {
	Reason string `json:"reason"`
}

func (h *Handler) AdminYankRelease(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	id, okID := parseUintParam(c, "id")
	if !okID {
		return
	}
	var req yankReleaseReq
	if !bindOptionalJSON(c, &req) {
		return
	}
	reason := truncate(strings.TrimSpace(req.Reason), 255)

	var release model.Release
	if err := h.db.First(&release, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, codeReleaseNotFound, fmt.Sprintf("release id=%d 不存在", id))
			return
		}
		failInternal(c, "admin: query release", err)
		return
	}
	if release.Status == model.ReleaseStatusYanked {
		ok(c, h.toReleaseAdminView(release)) // 幂等
		return
	}

	now := time.Now()
	if err := h.db.Model(&model.Release{}).Where("id = ?", release.ID).
		Updates(map[string]any{
			"status":      model.ReleaseStatusYanked,
			"yanked_at":   now,
			"yanked_by":   "admin",
			"yank_reason": reason,
		}).Error; err != nil {
		failInternal(c, "admin: yank release", err)
		return
	}
	release.Status = model.ReleaseStatusYanked
	release.YankedAt = &now
	release.YankedBy = "admin"
	release.YankReason = reason

	h.recordAudit(c, "admin", "", "release.yank", release.AppKey, "release",
		strconv.FormatUint(uint64(release.ID), 10),
		auditDetail(map[string]any{"versionCode": release.VersionCode, "reason": reason}))
	ok(c, h.toReleaseAdminView(release))
}

// AdminRestoreRelease POST /api/v1/admin/releases/:id/restore
func (h *Handler) AdminRestoreRelease(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	id, okID := parseUintParam(c, "id")
	if !okID {
		return
	}

	var release model.Release
	if err := h.db.First(&release, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, codeReleaseNotFound, fmt.Sprintf("release id=%d 不存在", id))
			return
		}
		failInternal(c, "admin: query release", err)
		return
	}
	if release.Status == model.ReleaseStatusLive {
		ok(c, h.toReleaseAdminView(release)) // 幂等
		return
	}

	if err := h.db.Model(&model.Release{}).Where("id = ?", release.ID).
		Updates(map[string]any{
			"status":      model.ReleaseStatusLive,
			"yanked_at":   nil,
			"yanked_by":   "",
			"yank_reason": "",
		}).Error; err != nil {
		failInternal(c, "admin: restore release", err)
		return
	}
	release.Status = model.ReleaseStatusLive
	release.YankedAt = nil
	release.YankedBy = ""
	release.YankReason = ""

	h.recordAudit(c, "admin", "", "release.restore", release.AppKey, "release",
		strconv.FormatUint(uint64(release.ID), 10),
		auditDetail(map[string]any{"versionCode": release.VersionCode}))
	ok(c, h.toReleaseAdminView(release))
}

// ---------- audit logs / usage ----------

// AdminListAuditLogs GET /api/v1/admin/audit-logs?appKey=&action=&limit=50
func (h *Handler) AdminListAuditLogs(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			fail(c, http.StatusBadRequest, codeInvalidParam, "limit 必须在 1..200 之间")
			return
		}
		limit = parsed
	}

	q := h.db.Model(&model.AuditLog{})
	if appKey := strings.TrimSpace(c.Query("appKey")); appKey != "" {
		q = q.Where("app_key = ?", appKey)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		q = q.Where("action = ?", action)
	}
	var entries []model.AuditLog
	if err := q.Order("id desc").Limit(limit).Find(&entries).Error; err != nil {
		failInternal(c, "admin: list audit logs", err)
		return
	}
	items := make([]auditLogView, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toAuditLogView(entry))
	}
	ok(c, gin.H{"items": items})
}

// AdminAppUsage GET /api/v1/admin/apps/:appKey/usage
func (h *Handler) AdminAppUsage(c *gin.Context) {
	app, found := h.adminApp(c)
	if !found {
		return
	}
	usage := usageView{AppKey: app.AppKey}
	start := startOfToday()

	if err := h.db.Model(&model.Release{}).
		Where("app_key = ? AND created_at >= ?", app.AppKey, start).
		Count(&usage.UploadsToday).Error; err != nil {
		failInternal(c, "admin: usage uploads today", err)
		return
	}
	if err := h.db.Model(&model.Release{}).
		Where("app_key = ?", app.AppKey).
		Count(&usage.ReleaseCount).Error; err != nil {
		failInternal(c, "admin: usage release count", err)
		return
	}
	if err := h.db.Model(&model.Release{}).
		Where("app_key = ? AND status = ?", app.AppKey, model.ReleaseStatusLive).
		Count(&usage.LiveReleaseCount).Error; err != nil {
		failInternal(c, "admin: usage live release count", err)
		return
	}
	if err := h.db.Model(&model.AppUploadKey{}).
		Where("app_id = ?", app.ID).
		Count(&usage.KeyCount).Error; err != nil {
		failInternal(c, "admin: usage key count", err)
		return
	}
	if err := h.db.Model(&model.AppUploadKey{}).
		Where("app_id = ? AND status = ?", app.ID, model.UploadKeyStatusActive).
		Count(&usage.ActiveKeyCount).Error; err != nil {
		failInternal(c, "admin: usage active key count", err)
		return
	}

	var latest model.Release
	err := h.db.Where("app_key = ? AND status = ?", app.AppKey, model.ReleaseStatusLive).
		Order("version_code desc").First(&latest).Error
	if err == nil {
		usage.LatestVersionCode = latest.VersionCode
		usage.LatestVersionName = latest.VersionName
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		failInternal(c, "admin: usage latest release", err)
		return
	}

	var last model.Release
	err = h.db.Where("app_key = ?", app.AppKey).Order("created_at desc").First(&last).Error
	if err == nil {
		uploadedAt := last.CreatedAt
		usage.LastUploadAt = &uploadedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		failInternal(c, "admin: usage last upload", err)
		return
	}

	ok(c, usage)
}

// parseBoolLoose 只把明确的真值字符串视为 true（includeYanked=1/true/yes）。
func parseBoolLoose(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
