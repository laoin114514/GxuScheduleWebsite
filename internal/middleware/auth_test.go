package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newAdminTestRouter(keys []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/ping", AdminAuth(keys), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": IsAdmin(c)})
	})
	return r
}

func doAdminReq(r *gin.Engine, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAdminAuth(t *testing.T) {
	keys := []string{"admin-key-one-0123456789", "admin-key-two-0123456789"}
	r := newAdminTestRouter(keys)

	if w := doAdminReq(r, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("no key = %d, want 401", w.Code)
	}
	if w := doAdminReq(r, map[string]string{AdminKeyHeader: "wrong"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key = %d, want 401", w.Code)
	}
	if w := doAdminReq(r, map[string]string{"Authorization": "Bearer admin-key-two-0123456789"}); w.Code != http.StatusOK {
		t.Fatalf("bearer second key = %d, want 200", w.Code)
	}
	if w := doAdminReq(r, map[string]string{AdminKeyHeader: "admin-key-one-0123456789"}); w.Code != http.StatusOK {
		t.Fatalf("x-admin-key = %d, want 200", w.Code)
	}
	// 非 Bearer 的 Authorization 不会被当成管理 key。
	if w := doAdminReq(r, map[string]string{"Authorization": "Basic abc"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("basic auth = %d, want 401", w.Code)
	}
}

func TestAdminAuthEmptyConfig(t *testing.T) {
	r := newAdminTestRouter(nil)
	if w := doAdminReq(r, nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty config = %d, want 503", w.Code)
	}
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, RequestIDFromContext(c))
	})

	// 生成并回填响应头。
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("response should carry generated X-Request-ID")
	}

	// 合法的客户端请求 ID 会被透传。
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "trace-abc_123")
	r.ServeHTTP(w, req)
	if got := w.Body.String(); got != "trace-abc_123" {
		t.Fatalf("context request id = %q, want trace-abc_123", got)
	}

	// 非法请求 ID（含空格/超长）不会被透传，防止日志注入。
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "bad value with spaces")
	r.ServeHTTP(w, req)
	if w.Body.String() == "bad value with spaces" {
		t.Fatal("invalid request id should be replaced")
	}
}
