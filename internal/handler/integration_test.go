package handler_test

// 真实 MySQL 的端到端集成测试：验证「admin 建 app/发 key → key 上传 → latest → 轮换/吊销/
// 配额/降级/yank/停用 → 审计」整条链路。对象存储用内存 fake，不连阿里云 OSS。
//
// 运行方式（需要可写的测试库，不会碰生产库）：
//
//	GXU_TEST_MYSQL_DSN='root:testroot@tcp(127.0.0.1:13306)/schedule_test?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test ./internal/handler/ -run TestPerAppUploadKeysEndToEnd -v
//
// 未设置 GXU_TEST_MYSQL_DSN 时自动 skip，普通 go test ./... 不受影响。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/laoin114514/gxuschedule-server/internal/auth"
	"github.com/laoin114514/gxuschedule-server/internal/config"
	"github.com/laoin114514/gxuschedule-server/internal/handler"
	"github.com/laoin114514/gxuschedule-server/internal/model"
	"github.com/laoin114514/gxuschedule-server/internal/router"
)

const (
	testAdminKey   = "admin-key-0123456789abcdef"
	testDSNEnvName = "GXU_TEST_MYSQL_DSN"
	testAppKey     = "schedule"
)

// ---------- fake 对象存储 ----------

type fakeStore struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func newFakeStore() *fakeStore {
	return &fakeStore{objs: make(map[string][]byte)}
}

func (f *fakeStore) PutObject(key, localPath string, size int64, contentType string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objs[key] = data
	return nil
}

func (f *fakeStore) URL(key string) (string, error) {
	return "https://cdn.test/" + key, nil
}

func (f *fakeStore) DeleteObject(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objs, key)
	return nil
}

func (f *fakeStore) object(key string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objs[key]
	return data, ok
}

// ---------- 测试基座 ----------

func testConfig() *config.Config {
	return &config.Config{
		AppEnv:                  "test",
		AdminAPIKeys:            []string{testAdminKey},
		MaxUploadMB:             512,
		DefaultDailyUploadLimit: 20,
		RateLimitRPS:            1000,
		RateLimitBurst:          1000,
		UploadRateRPS:           1000,
		UploadRateBurst:         1000,
		AdminRateRPS:            1000,
		AdminRateBurst:          1000,
		CORSAllowOrigins:        []string{"*"},
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv(testDSNEnvName)
	if dsn == "" {
		t.Skipf("跳过集成测试：未设置 %s", testDSNEnvName)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect test mysql: %v", err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.AppUploadKey{}, &model.Release{}, &model.AuditLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	for _, table := range []string{"app_upload_keys", "releases", "audit_logs", "apps"} {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	return db
}

func newEngine(t *testing.T, db *gorm.DB, cfg *config.Config, store *fakeStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := handler.NewWithStorage(db, store, cfg, auth.NewDBResolver(db))
	return router.New(cfg, h)
}

// ---------- HTTP 辅助 ----------

type apiResponse struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	ErrorCode string          `json:"errorCode"`
	Data      json.RawMessage `json:"data"`
}

func doRequest(t *testing.T, r *gin.Engine, method, path string, headers map[string]string, body []byte, contentType string) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp apiResponse
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s %s: response not JSON: %v, body=%q", method, path, err, w.Body.String())
		}
	}
	return w, resp
}

func adminCall(t *testing.T, r *gin.Engine, method, path string, body any) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal admin body: %v", err)
		}
	}
	return doRequest(t, r, method, path,
		map[string]string{"Authorization": "Bearer " + testAdminKey}, raw, "application/json")
}

type uploadOptions struct {
	apiKey   string
	appKey   string // 空字符串表示不传 appKey 字段
	version  int64
	name     string
	fileName string
	content  []byte
	extra    map[string]string
}

func uploadRelease(t *testing.T, r *gin.Engine, opt uploadOptions) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	writeField := func(key, value string) {
		if err := mw.WriteField(key, value); err != nil {
			t.Fatalf("write field %s: %v", key, err)
		}
	}
	if opt.appKey != "" {
		writeField("appKey", opt.appKey)
	}
	writeField("versionCode", fmt.Sprintf("%d", opt.version))
	writeField("versionName", opt.name)
	for k, v := range opt.extra {
		writeField(k, v)
	}
	fw, err := mw.CreateFormFile("file", opt.fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(opt.content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return doRequest(t, r, http.MethodPost, "/api/v1/releases/upload",
		map[string]string{"X-API-Key": opt.apiKey}, buf.Bytes(), mw.FormDataContentType())
}

func wantOK(t *testing.T, label string, w *httptest.ResponseRecorder, resp apiResponse) {
	t.Helper()
	if w.Code != http.StatusOK || resp.Code != 0 {
		t.Fatalf("%s: status=%d code=%d body=%s", label, w.Code, resp.Code, w.Body.String())
	}
}

func wantErr(t *testing.T, label string, w *httptest.ResponseRecorder, resp apiResponse, status int, errorCode string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("%s: status=%d, want %d, body=%s", label, w.Code, status, w.Body.String())
	}
	if resp.Code != status {
		t.Fatalf("%s: body code=%d, want %d, body=%s", label, resp.Code, status, w.Body.String())
	}
	if resp.ErrorCode != errorCode {
		t.Fatalf("%s: errorCode=%q, want %q, body=%s", label, resp.ErrorCode, errorCode, w.Body.String())
	}
}

func decodeData[T any](t *testing.T, label string, resp apiResponse) T {
	t.Helper()
	var out T
	if len(resp.Data) == 0 {
		t.Fatalf("%s: empty data, body=%s", label, string(resp.Data))
	}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		t.Fatalf("%s: unmarshal data: %v, data=%s", label, err, string(resp.Data))
	}
	return out
}

// ---------- 响应结构 ----------

type keyIssue struct {
	KeyID     string `json:"keyId"`
	Plaintext string `json:"plaintext"`
	Last4     string `json:"last4"`
	Label     string `json:"label"`
	Status    int8   `json:"status"`
}

type createAppResult struct {
	App struct {
		AppKey string `json:"appKey"`
		Name   string `json:"name"`
		Status int8   `json:"status"`
	} `json:"app"`
	UploadKey *keyIssue `json:"uploadKey"`
}

type uploadResult struct {
	ReleaseID   uint   `json:"releaseId"`
	AppKey      string `json:"appKey"`
	VersionCode int64  `json:"versionCode"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
}

type latestResult struct {
	AppKey             string `json:"appKey"`
	HasUpdate          bool   `json:"hasUpdate"`
	LatestVersionCode  int64  `json:"latestVersionCode"`
	LatestVersionName  string `json:"latestVersionName"`
	CurrentVersionCode int64  `json:"currentVersionCode"`
	DownloadURL        string `json:"downloadUrl"`
}

type listKeysResult struct {
	Items []struct {
		KeyID  string `json:"keyId"`
		Status int8   `json:"status"`
		Last4  string `json:"last4"`
	} `json:"items"`
}

type rotateResult struct {
	UploadKey keyIssue `json:"uploadKey"`
	RetireKey struct {
		KeyID  string `json:"keyId"`
		Status int8   `json:"status"`
	} `json:"retireKey"`
}

type listReleasesResult struct {
	Items []struct {
		ReleaseID   uint  `json:"releaseId"`
		VersionCode int64 `json:"versionCode"`
		Status      int8  `json:"status"`
	} `json:"items"`
}

type auditResult struct {
	Items []struct {
		Action    string `json:"action"`
		ActorType string `json:"actorType"`
		AppKey    string `json:"appKey"`
	} `json:"items"`
}

// ---------- 完整链路测试 ----------

func TestPerAppUploadKeysEndToEnd(t *testing.T) {
	db := openTestDB(t)
	store := newFakeStore()
	cfg := testConfig()
	r := newEngine(t, db, cfg, store)

	var (
		firstKey    keyIssue
		firstRelID  uint
		release99ID uint
	)

	t.Run("admin 建 app 并签发首把 key", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPost, "/api/v1/admin/apps", map[string]any{
			"appKey": testAppKey, "name": "西大课栈", "label": "gha-release",
		})
		wantOK(t, "create app", w, resp)
		result := decodeData[createAppResult](t, "create app", resp)
		if result.App.AppKey != testAppKey || result.UploadKey == nil {
			t.Fatalf("create app result = %+v", result)
		}
		if len(result.UploadKey.Plaintext) < 20 || result.UploadKey.Plaintext[:len(auth.KeyPrefix)] != auth.KeyPrefix {
			t.Fatalf("unexpected plaintext: %q", result.UploadKey.Plaintext)
		}
		firstKey = *result.UploadKey

		var dbKey model.AppUploadKey
		if err := db.Where("key_id = ?", firstKey.KeyID).First(&dbKey).Error; err != nil {
			t.Fatalf("query issued key: %v", err)
		}
		if dbKey.KeyHash != auth.HashKey(firstKey.Plaintext) {
			t.Fatal("db key hash mismatch")
		}
		if dbKey.Status != model.UploadKeyStatusActive {
			t.Fatalf("key status = %d, want active", dbKey.Status)
		}
	})

	t.Run("重复建 app 409、重复发 active key 409", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPost, "/api/v1/admin/apps", map[string]any{
			"appKey": testAppKey, "name": "重复",
		})
		wantErr(t, "duplicate app", w, resp, http.StatusConflict, "APP_EXISTS")

		w, resp = adminCall(t, r, http.MethodPost, "/api/v1/admin/apps/"+testAppKey+"/keys", map[string]any{"label": "second"})
		wantErr(t, "second active key", w, resp, http.StatusConflict, "ACTIVE_KEY_EXISTS")
	})

	t.Run("上传 + 幂等 + 冲突 + 降级", func(t *testing.T) {
		w, resp := uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 100, name: "1.0.0",
			fileName: "app-universal.apk", content: []byte("APK-CONTENT-100"),
		})
		wantOK(t, "upload v100", w, resp)
		uploaded := decodeData[uploadResult](t, "upload v100", resp)
		if uploaded.ReleaseID == 0 || uploaded.VersionCode != 100 || uploaded.DownloadURL == "" {
			t.Fatalf("upload result = %+v", uploaded)
		}
		firstRelID = uploaded.ReleaseID
		if data, ok := store.object("release/schedule/100/app-universal.apk"); !ok || !bytes.Equal(data, []byte("APK-CONTENT-100")) {
			t.Fatalf("fake store object missing/wrong: %q", data)
		}

		// 幂等：同版本同内容重复上传 → 200 同 releaseId，且不占配额。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 100, name: "1.0.0",
			fileName: "app-universal.apk", content: []byte("APK-CONTENT-100"),
		})
		wantOK(t, "idempotent upload", w, resp)
		idem := decodeData[uploadResult](t, "idempotent upload", resp)
		if idem.ReleaseID != firstRelID {
			t.Fatalf("idempotent releaseId = %d, want %d", idem.ReleaseID, firstRelID)
		}

		// 同版本不同内容 → 409。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 100, name: "1.0.0",
			fileName: "app-universal.apk", content: []byte("DIFFERENT-CONTENT"),
		})
		wantErr(t, "conflict upload", w, resp, http.StatusConflict, "VERSION_CONFLICT")

		// 低于最新 → 409 DOWNGRADE_REJECTED。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 99, name: "0.9.9",
			fileName: "app-99.apk", content: []byte("APK-CONTENT-99"),
		})
		wantErr(t, "downgrade rejected", w, resp, http.StatusConflict, "DOWNGRADE_REJECTED")

		// 开 allowDowngrade 后允许。
		w, resp = adminCall(t, r, http.MethodPatch, "/api/v1/admin/apps/"+testAppKey, map[string]any{"allowDowngrade": true})
		wantOK(t, "patch allowDowngrade", w, resp)
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 99, name: "0.9.9",
			fileName: "app-99.apk", content: []byte("APK-CONTENT-99"),
		})
		wantOK(t, "upload v99 after allowDowngrade", w, resp)
		release99ID = decodeData[uploadResult](t, "upload v99", resp).ReleaseID
	})

	t.Run("latest / appKey 不匹配 / 未知 app", func(t *testing.T) {
		w, resp := doRequest(t, r, http.MethodGet, "/api/v1/apps/"+testAppKey+"/latest?versionCode=99", nil, nil, "")
		wantOK(t, "latest", w, resp)
		latest := decodeData[latestResult](t, "latest", resp)
		if !latest.HasUpdate || latest.LatestVersionCode != 100 {
			t.Fatalf("latest = %+v, want hasUpdate + latest=100", latest)
		}

		// key 属于 schedule，表单却写别的 appKey → 403。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: "another-app", version: 101, name: "1.0.1",
			fileName: "app-101.apk", content: []byte("APK-101"),
		})
		wantErr(t, "appKey mismatch", w, resp, http.StatusForbidden, "APP_KEY_MISMATCH")

		// 未知 app 保持 200 空（官网兼容）。
		w, resp = doRequest(t, r, http.MethodGet, "/api/v1/apps/no-such-app/latest?versionCode=0", nil, nil, "")
		wantOK(t, "latest unknown app", w, resp)
		unknown := decodeData[latestResult](t, "latest unknown app", resp)
		if unknown.HasUpdate || unknown.LatestVersionCode != 0 {
			t.Fatalf("unknown app latest = %+v, want empty", unknown)
		}
	})

	t.Run("每日配额", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPatch, "/api/v1/admin/apps/"+testAppKey, map[string]any{"dailyUploadLimit": 2})
		wantOK(t, "patch daily limit", w, resp)
		// 今天已有 2 条（v100、v99），再传新版本触发 429。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 103, name: "1.0.3",
			fileName: "app-103.apk", content: []byte("APK-103"),
		})
		wantErr(t, "daily quota", w, resp, http.StatusTooManyRequests, "UPLOAD_QUOTA_EXCEEDED")
		w, resp = adminCall(t, r, http.MethodPatch, "/api/v1/admin/apps/"+testAppKey, map[string]any{"dailyUploadLimit": 20})
		wantOK(t, "restore daily limit", w, resp)
	})

	var rotated keyIssue
	t.Run("轮换：旧 key 宽限期仍可用、每 app 仅一把 active", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPost, "/api/v1/admin/keys/"+firstKey.KeyID+"/rotate",
			map[string]any{"graceHours": 2, "label": "gha-release-v2"})
		wantOK(t, "rotate", w, resp)
		rotatedResult := decodeData[rotateResult](t, "rotate", resp)
		if rotatedResult.UploadKey.Plaintext == "" || rotatedResult.RetireKey.KeyID != firstKey.KeyID {
			t.Fatalf("rotate result = %+v", rotatedResult)
		}
		if rotatedResult.RetireKey.Status != model.UploadKeyStatusRetiring {
			t.Fatalf("retire key status = %d, want retiring", rotatedResult.RetireKey.Status)
		}
		rotated = rotatedResult.UploadKey

		var activeCount int64
		if err := db.Model(&model.AppUploadKey{}).
			Where("status = ?", model.UploadKeyStatusActive).Count(&activeCount).Error; err != nil {
			t.Fatalf("count active keys: %v", err)
		}
		if activeCount != 1 {
			t.Fatalf("active key count = %d, want 1", activeCount)
		}

		// 旧 key（retiring）在宽限期内仍可上传新版本。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: firstKey.Plaintext, appKey: testAppKey, version: 102, name: "1.0.2",
			fileName: "app-102.apk", content: []byte("APK-102"),
		})
		wantOK(t, "upload with retiring key", w, resp)

		// 新 key 也能上传。
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: rotated.Plaintext, appKey: testAppKey, version: 104, name: "1.0.4",
			fileName: "app-104.apk", content: []byte("APK-104"),
		})
		wantOK(t, "upload with rotated key", w, resp)
	})

	t.Run("吊销后 401、过期 key 401、可重新签发", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPost, "/api/v1/admin/keys/"+rotated.KeyID+"/revoke",
			map[string]any{"reason": "test revoke"})
		wantOK(t, "revoke", w, resp)
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: rotated.Plaintext, appKey: testAppKey, version: 105, name: "1.0.5",
			fileName: "app-105.apk", content: []byte("APK-105"),
		})
		wantErr(t, "upload with revoked key", w, resp, http.StatusUnauthorized, "INVALID_API_KEY")

		// 过期 key 同样 401（签发时给过去时间）。
		w, resp = adminCall(t, r, http.MethodPost, "/api/v1/admin/apps/"+testAppKey+"/keys",
			map[string]any{"label": "expired", "expiresAt": "2020-01-01T00:00:00Z"})
		wantOK(t, "issue expired key", w, resp)
		expired := decodeData[keyIssue](t, "issue expired key", resp)
		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: expired.Plaintext, appKey: testAppKey, version: 106, name: "1.0.6",
			fileName: "app-106.apk", content: []byte("APK-106"),
		})
		wantErr(t, "upload with expired key", w, resp, http.StatusUnauthorized, "INVALID_API_KEY")

		// 吊销过期 key 后可以再发新的 active key。
		_, _ = adminCall(t, r, http.MethodPost, "/api/v1/admin/keys/"+expired.KeyID+"/revoke", nil)
		w, resp = adminCall(t, r, http.MethodPost, "/api/v1/admin/apps/"+testAppKey+"/keys",
			map[string]any{"label": "after-revoke"})
		wantOK(t, "issue key after revoke", w, resp)
		rotated = decodeData[keyIssue](t, "issue key after revoke", resp)
	})

	t.Run("yank 隐藏最新版并回退、restore 恢复", func(t *testing.T) {
		// 当前最新是 104（v102、v104 都上传成功）。
		var latestRel model.Release
		if err := db.Where("app_key = ? AND version_code = ?", testAppKey, 104).First(&latestRel).Error; err != nil {
			t.Fatalf("query v104: %v", err)
		}
		w, resp := adminCall(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/releases/%d", latestRel.ID),
			map[string]any{"reason": "test yank"})
		wantOK(t, "yank", w, resp)

		w, resp = doRequest(t, r, http.MethodGet, "/api/v1/apps/"+testAppKey+"/latest?versionCode=0", nil, nil, "")
		wantOK(t, "latest after yank", w, resp)
		afterYank := decodeData[latestResult](t, "latest after yank", resp)
		if afterYank.LatestVersionCode != 102 {
			t.Fatalf("latest after yank = %d, want 102", afterYank.LatestVersionCode)
		}

		w, resp = adminCall(t, r, http.MethodPost, fmt.Sprintf("/api/v1/admin/releases/%d/restore", latestRel.ID), nil)
		wantOK(t, "restore", w, resp)
		w, resp = doRequest(t, r, http.MethodGet, "/api/v1/apps/"+testAppKey+"/latest?versionCode=0", nil, nil, "")
		wantOK(t, "latest after restore", w, resp)
		afterRestore := decodeData[latestResult](t, "latest after restore", resp)
		if afterRestore.LatestVersionCode != 104 {
			t.Fatalf("latest after restore = %d, want 104", afterRestore.LatestVersionCode)
		}
		_ = release99ID
	})

	t.Run("停用 app：上传 403、latest 404，重新启用恢复", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodPatch, "/api/v1/admin/apps/"+testAppKey, map[string]any{"status": 2})
		wantOK(t, "disable app", w, resp)

		w, resp = uploadRelease(t, r, uploadOptions{
			apiKey: rotated.Plaintext, appKey: testAppKey, version: 107, name: "1.0.7",
			fileName: "app-107.apk", content: []byte("APK-107"),
		})
		wantErr(t, "upload to disabled app", w, resp, http.StatusForbidden, "APP_DISABLED")

		w, resp = doRequest(t, r, http.MethodGet, "/api/v1/apps/"+testAppKey+"/latest?versionCode=0", nil, nil, "")
		wantErr(t, "latest disabled app", w, resp, http.StatusNotFound, "APP_NOT_FOUND")

		w, resp = adminCall(t, r, http.MethodPatch, "/api/v1/admin/apps/"+testAppKey, map[string]any{"status": 1})
		wantOK(t, "enable app", w, resp)
	})

	t.Run("审计日志记录关键动作", func(t *testing.T) {
		w, resp := adminCall(t, r, http.MethodGet, "/api/v1/admin/audit-logs?appKey="+testAppKey+"&limit=200", nil)
		wantOK(t, "audit logs", w, resp)
		audit := decodeData[auditResult](t, "audit logs", resp)
		if len(audit.Items) == 0 {
			t.Fatal("audit logs empty")
		}
		actions := make(map[string]bool, len(audit.Items))
		for _, item := range audit.Items {
			actions[item.Action] = true
		}
		for _, want := range []string{"app.create", "key.issue", "key.rotate", "key.revoke", "release.yank", "release.restore", "upload.success"} {
			if !actions[want] {
				t.Fatalf("audit logs missing action %q, got %v", want, actions)
			}
		}
	})
}
