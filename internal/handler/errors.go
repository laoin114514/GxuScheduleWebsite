package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 稳定错误码：CI/前端/运维脚本依赖这些字符串，提示文案可以改，错误码不要改。
const (
	codeInvalidAPIKey       = "INVALID_API_KEY"
	codeAppNotFound         = "APP_NOT_FOUND"
	codeAppDisabled         = "APP_DISABLED"
	codeAppKeyMismatch      = "APP_KEY_MISMATCH"
	codeInvalidParam        = "INVALID_PARAM"
	codeFileTooLarge        = "FILE_TOO_LARGE"
	codeVersionConflict     = "VERSION_CONFLICT"
	codeDowngradeRejected   = "DOWNGRADE_REJECTED"
	codeUploadQuotaExceeded = "UPLOAD_QUOTA_EXCEEDED"
	codeRateLimited         = "RATE_LIMITED"
	codeAppExists           = "APP_EXISTS"
	codeKeyNotFound         = "KEY_NOT_FOUND"
	codeKeyNotActive        = "KEY_NOT_ACTIVE"
	codeActiveKeyExists     = "ACTIVE_KEY_EXISTS"
	codeReleaseNotFound     = "RELEASE_NOT_FOUND"
	codeInternal            = "INTERNAL"
	codeOSSError            = "OSS_ERROR"
)

// ok 成功响应：{"code":0,"message":"ok","data":...}（保持旧客户端契约）。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// fail 失败响应：{"code":<HTTP 状态码>,"errorCode":"<稳定大写下划线码>","message":"<中文提示>"}。
func fail(c *gin.Context, status int, errorCode, message string) {
	c.JSON(status, gin.H{"code": status, "errorCode": errorCode, "message": message})
}

// failInternal 把真实错误写服务端日志，对外只返回统一的 500（不泄露内部细节）。
func failInternal(c *gin.Context, action string, err error) {
	log.Printf("%s: %v", action, err)
	fail(c, http.StatusInternalServerError, codeInternal, "服务内部错误，请稍后重试")
}
