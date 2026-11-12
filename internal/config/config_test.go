package config

import (
	"strings"
	"testing"
)

// setBaseEnv 填入一组能通过校验的最小 env；测试结束自动恢复。
func setBaseEnv(t *testing.T) {
	t.Helper()
	envs := map[string]string{
		"ADMIN_API_KEYS":             "admin-key-0123456789abcdef",
		"UPLOAD_AUTH_MODE":           "",
		"UPLOAD_API_KEY":             "",
		"UPLOAD_LEGACY_APPKEYS":      "",
		"DB_HOST":                    "127.0.0.1",
		"DB_USER":                    "schedule",
		"DB_PASSWORD":                "change-me",
		"DB_NAME":                    "schedule",
		"OSS_ENDPOINT":               "oss-cn-shenzhen.aliyuncs.com",
		"OSS_ACCESS_KEY_ID":          "test-access-key-id",
		"OSS_ACCESS_KEY_SECRET":      "test-access-key-secret",
		"OSS_BUCKET":                 "test-bucket",
		"DEFAULT_DAILY_UPLOAD_LIMIT": "",
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func TestLoadScopedDefaults(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UploadAuthMode != UploadAuthScoped {
		t.Fatalf("mode = %q, want %q", cfg.UploadAuthMode, UploadAuthScoped)
	}
	if len(cfg.AdminAPIKeys) != 1 || cfg.AdminAPIKeys[0] != "admin-key-0123456789abcdef" {
		t.Fatalf("admin keys = %v", cfg.AdminAPIKeys)
	}
	if cfg.UsesLegacyUploadKey() {
		t.Fatal("scoped mode must not use legacy key")
	}
	if !cfg.LegacyUploadAllows("schedule") {
		t.Fatal("empty legacy allowlist should allow any appKey")
	}
	if cfg.DefaultDailyUploadLimit != 20 {
		t.Fatalf("default daily limit = %d, want 20", cfg.DefaultDailyUploadLimit)
	}
	if cfg.UploadRateRPS <= 0 || cfg.UploadRateBurst <= 0 {
		t.Fatalf("upload rate defaults invalid: %v/%v", cfg.UploadRateRPS, cfg.UploadRateBurst)
	}
}

func TestLoadMultipleAdminKeys(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("ADMIN_API_KEYS", "first-key-0123456789, second-key-0123456789")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.AdminAPIKeys) != 2 {
		t.Fatalf("admin keys = %v, want 2", cfg.AdminAPIKeys)
	}
}

func TestLoadDualMode(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("UPLOAD_AUTH_MODE", "dual")
	t.Setenv("UPLOAD_API_KEY", "legacy-key-0123456789abcdef")
	t.Setenv("UPLOAD_LEGACY_APPKEYS", "schedule, schedule-beta")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UploadAuthMode != UploadAuthDual {
		t.Fatalf("mode = %q, want dual", cfg.UploadAuthMode)
	}
	if !cfg.UsesLegacyUploadKey() {
		t.Fatal("dual mode should use legacy key")
	}
	if !cfg.LegacyUploadAllows("schedule-beta") {
		t.Fatal("schedule-beta should be allowed by whitelist")
	}
	if cfg.LegacyUploadAllows("other-app") {
		t.Fatal("other-app should be rejected by whitelist")
	}

	t.Setenv("UPLOAD_AUTH_MODE", "legacy")
	if _, err := Load(); err != nil {
		t.Fatalf("legacy mode Load: %v", err)
	}
}

func TestLoadRejectsDualWithoutLegacyKey(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("UPLOAD_AUTH_MODE", "dual")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "UPLOAD_API_KEY") {
		t.Fatalf("err = %v, want mention of UPLOAD_API_KEY", err)
	}
}

func TestLoadRejectsMissingAdminKeys(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("ADMIN_API_KEYS", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_API_KEYS") {
		t.Fatalf("err = %v, want mention of ADMIN_API_KEYS", err)
	}
}

func TestLoadRejectsShortAdminKey(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("ADMIN_API_KEYS", "short")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "过短") {
		t.Fatalf("err = %v, want short-key error", err)
	}
}

func TestLoadRejectsAdminKeyEqualsLegacy(t *testing.T) {
	setBaseEnv(t)
	same := "same-key-0123456789abcdef"
	t.Setenv("ADMIN_API_KEYS", same)
	t.Setenv("UPLOAD_AUTH_MODE", "dual")
	t.Setenv("UPLOAD_API_KEY", same)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "不能与") {
		t.Fatalf("err = %v, want admin/legacy duplicate error", err)
	}
}

func TestLoadRejectsInvalidMode(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("UPLOAD_AUTH_MODE", "wild")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "UPLOAD_AUTH_MODE") {
		t.Fatalf("err = %v, want mode error", err)
	}
}
