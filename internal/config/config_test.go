package config

import (
	"strings"
	"testing"
)

// setBaseEnv 填入一组能通过校验的最小 env；测试结束自动恢复。
func setBaseEnv(t *testing.T) {
	t.Helper()
	envs := map[string]string{
		"ADMIN_API_KEYS":        "admin-key-0123456789abcdef",
		"DB_HOST":               "127.0.0.1",
		"DB_USER":               "schedule",
		"DB_PASSWORD":           "change-me",
		"DB_NAME":               "schedule",
		"OSS_ENDPOINT":          "oss-cn-shenzhen.aliyuncs.com",
		"OSS_ACCESS_KEY_ID":     "test-access-key-id",
		"OSS_ACCESS_KEY_SECRET": "test-access-key-secret",
		"OSS_BUCKET":            "test-bucket",
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.AdminAPIKeys) != 1 || cfg.AdminAPIKeys[0] != "admin-key-0123456789abcdef" {
		t.Fatalf("admin keys = %v", cfg.AdminAPIKeys)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("AppEnv = %q, want development", cfg.AppEnv)
	}
	if cfg.MaxUploadMB != 512 || cfg.DefaultDailyUploadLimit != 20 {
		t.Fatalf("defaults = %d/%d", cfg.MaxUploadMB, cfg.DefaultDailyUploadLimit)
	}
	if cfg.UploadRateRPS <= 0 || cfg.AdminRateRPS <= 0 {
		t.Fatalf("rate defaults invalid: upload=%v admin=%v", cfg.UploadRateRPS, cfg.AdminRateRPS)
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

func TestLoadRejectsMissingDBAndOSS(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DB_USER", "")
	t.Setenv("OSS_BUCKET", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DB_USER") || !strings.Contains(err.Error(), "OSS_BUCKET") {
		t.Fatalf("err = %v, want DB_USER/OSS_BUCKET errors", err)
	}
}
