package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/laoin114514/gxuschedule-server/internal/config"
	"github.com/laoin114514/gxuschedule-server/internal/handler"
)

const testAdminKey = "admin-test-key-0123456789"

// newTestRouter 只验证路由注册与中间件（鉴权/限流/参数校验）。
// handler 的 db/oss 均为 nil：只要用例不构造需要读库成功路径的请求，
// 处理器必须安全地返回错误响应（不 panic）。
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		AppEnv:                  "test",
		AdminAPIKeys:            []string{testAdminKey},
		RateLimitRPS:            0.5,
		RateLimitBurst:          3,
		UploadRateRPS:           1,
		UploadRateBurst:         5,
		AdminRateRPS:            1,
		AdminRateBurst:          5,
		MaxUploadMB:             512,
		DefaultDailyUploadLimit: 20,
	}
	return New(cfg, handler.New(nil, nil, cfg))
}

type testRequest struct {
	method  string
	path    string
	headers map[string]string
	body    string
}

func doReq(t *testing.T, r *gin.Engine, req testRequest) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if req.body != "" {
		body = strings.NewReader(req.body)
	}
	httpReq := httptest.NewRequest(req.method, req.path, body)
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not JSON: %v, body=%q", err, w.Body.String())
	}
	return payload
}

// wantError 断言 HTTP 状态码、统一错误响应格式（code=status）与稳定 errorCode。
func wantError(t *testing.T, label string, w *httptest.ResponseRecorder, status int, errorCode string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("%s: status = %d, want %d, body=%s", label, w.Code, status, w.Body.String())
	}
	payload := decodeJSON(t, w)
	if got := payload["errorCode"]; got != errorCode {
		t.Fatalf("%s: errorCode = %v, want %s, body=%s", label, got, errorCode, w.Body.String())
	}
	code, ok := payload["code"].(float64)
	if !ok || int(code) != status {
		t.Fatalf("%s: code = %v, want %d, body=%s", label, payload["code"], status, w.Body.String())
	}
	if _, ok := payload["message"].(string); !ok {
		t.Fatalf("%s: missing message, body=%s", label, w.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	r := newTestRouter(t)
	w := doReq(t, r, testRequest{method: http.MethodGet, path: "/healthz"})
	if w.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	payload := decodeJSON(t, w)
	if payload["status"] != "ok" {
		t.Fatalf("healthz status = %v, want ok", payload["status"])
	}
	if _, ok := payload["web"]; !ok {
		t.Fatalf("healthz missing web field, body=%s", w.Body.String())
	}
}

func TestUploadRejectsMissingKey(t *testing.T) {
	r := newTestRouter(t)
	w := doReq(t, r, testRequest{method: http.MethodPost, path: "/api/v1/releases/upload"})
	wantError(t, "upload without key", w, http.StatusUnauthorized, "INVALID_API_KEY")
}

func TestUploadRejectsInvalidKey(t *testing.T) {
	r := newTestRouter(t)
	for _, key := range []string{
		"not-a-valid-key",
		"gxu_live_00000000_" + strings.Repeat("A", 43), // 格式合法但库里不存在（nil db 同样找不到）
	} {
		w := doReq(t, r, testRequest{
			method:  http.MethodPost,
			path:    "/api/v1/releases/upload",
			headers: map[string]string{"X-API-Key": key},
		})
		wantError(t, "upload with invalid key", w, http.StatusUnauthorized, "INVALID_API_KEY")
	}
}

// admin key 绝不能当上传 key 用：即使它是合法管理密钥也必须 401，且错误码不是 ADMIN_UNAUTHORIZED。
func TestUploadRejectsAdminKey(t *testing.T) {
	r := newTestRouter(t)
	w := doReq(t, r, testRequest{
		method:  http.MethodPost,
		path:    "/api/v1/releases/upload",
		headers: map[string]string{"X-API-Key": testAdminKey},
	})
	wantError(t, "upload with admin key", w, http.StatusUnauthorized, "INVALID_API_KEY")
}

func TestAdminAuth(t *testing.T) {
	r := newTestRouter(t)

	if w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/admin/apps"}); w.Code != http.StatusUnauthorized {
		wantError(t, "admin without key", w, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED")
	}
	if w := doReq(t, r, testRequest{
		method:  http.MethodGet,
		path:    "/api/v1/admin/apps",
		headers: map[string]string{"X-Admin-Key": "wrong-admin-key"},
	}); w.Code != http.StatusUnauthorized {
		wantError(t, "admin with wrong key", w, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED")
	}
	// 上传 key 没有管理权限（X-API-Key 不被 admin 鉴权读取）。
	if w := doReq(t, r, testRequest{
		method:  http.MethodGet,
		path:    "/api/v1/admin/apps",
		headers: map[string]string{"X-API-Key": testAdminKey},
	}); w.Code != http.StatusUnauthorized {
		wantError(t, "upload key as admin key", w, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED")
	}
	// 正确 admin key + nil db：处理器返回 500 INTERNAL（而不是 panic）。
	w := doReq(t, r, testRequest{
		method:  http.MethodGet,
		path:    "/api/v1/admin/apps",
		headers: map[string]string{"Authorization": "Bearer " + testAdminKey},
	})
	wantError(t, "admin with valid key, nil db", w, http.StatusInternalServerError, "INTERNAL")
}

func TestLatestParamValidation(t *testing.T) {
	r := newTestRouter(t)
	if w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/apps/BAD-KEY/latest?versionCode=1"}); w.Code != http.StatusBadRequest {
		wantError(t, "latest bad appKey", w, http.StatusBadRequest, "INVALID_PARAM")
	}
	if w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/apps/schedule/latest?versionCode=abc"}); w.Code != http.StatusBadRequest {
		wantError(t, "latest bad versionCode", w, http.StatusBadRequest, "INVALID_PARAM")
	}
	if w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/apps/schedule/latest?versionCode=-1"}); w.Code != http.StatusBadRequest {
		wantError(t, "latest negative versionCode", w, http.StatusBadRequest, "INVALID_PARAM")
	}
}

// db 为 nil 时最新版查询必须安全返回 500，而不是 panic。
func TestLatestNilDBSafe(t *testing.T) {
	r := newTestRouter(t)
	w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/apps/schedule/latest?versionCode=1"})
	wantError(t, "latest with nil db", w, http.StatusInternalServerError, "INTERNAL")
}

func TestLatestRateLimit(t *testing.T) {
	r := newTestRouter(t)
	got429 := false
	for i := 0; i < 10; i++ {
		// 前三发命中处理器（db=nil → 500），桶耗尽后由中间件返回 429。
		w := doReq(t, r, testRequest{method: http.MethodGet, path: "/api/v1/apps/schedule/latest?versionCode=1"})
		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("expect 429 after burst exhausted, got none")
	}
}
