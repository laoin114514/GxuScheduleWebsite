package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newStaticTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "index.html"), "<html>spa-index</html>")
	mustWrite(t, filepath.Join(dir, "assets", "app-abc123.js"), "console.log('ok')")

	r := gin.New()
	if !Register(r, dir) {
		t.Fatal("Register returned false for a valid dir")
	}
	return r
}

func mustWrite(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestStaticAssetServedWithImmutableCache(t *testing.T) {
	r := newStaticTestRouter(t)
	w := get(t, r, "/assets/app-abc123.js")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "console.log") {
		t.Fatalf("body = %q, want asset content", w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("Cache-Control = %q, want immutable", cc)
	}
}

// 发版瞬间 index.html 先到、新哈希资源还没上传时，缺失资源必须 404。
// 若回退 index.html，浏览器会把 HTML 当 JS 执行导致整页空白，CDN 也会错误缓存。
func TestMissingAssetReturns404NotIndex(t *testing.T) {
	r := newStaticTestRouter(t)
	w := get(t, r, "/assets/index-deadbeef.js")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if strings.Contains(w.Body.String(), "spa-index") {
		t.Fatal("missing asset fell back to index.html")
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
}

func TestSpaFallbackIsNotCached(t *testing.T) {
	r := newStaticTestRouter(t)
	w := get(t, r, "/some/client/route")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "spa-index") {
		t.Fatalf("body = %q, want index.html", w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
}

func TestUnknownApiReturnsJson404(t *testing.T) {
	r := newStaticTestRouter(t)
	w := get(t, r, "/api/v1/nope")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "接口不存在") {
		t.Fatalf("body = %q, want json error", w.Body.String())
	}
}
