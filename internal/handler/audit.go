package handler

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/laoin114514/gxuschedule-server/internal/middleware"
	"github.com/laoin114514/gxuschedule-server/internal/model"
)

// recordAudit 落一条审计日志。只记录元信息，绝不落明文 key。
//
// 审计不是业务链路的一部分：写失败只打日志，绝不影响请求结果；
// db 为 nil（纯路由/鉴权单测）时直接跳过。
func (h *Handler) recordAudit(c *gin.Context, actorType, actorKeyID, action, appKey, targetType, targetID, detail string) {
	if h.db == nil {
		return
	}
	if detail == "" {
		detail = "{}"
	}
	entry := model.AuditLog{
		RequestID:  middleware.RequestIDFromContext(c),
		ActorType:  actorType,
		ActorKeyID: actorKeyID,
		Action:     action,
		AppKey:     appKey,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         c.ClientIP(),
		UserAgent:  truncate(c.Request.UserAgent(), 255),
		Detail:     detail,
	}
	if err := h.db.Create(&entry).Error; err != nil {
		log.Printf("audit: write failed action=%s appKey=%s: %v", action, appKey, err)
	}
}

// auditDetail 把键值序列化成 JSON 文本（AuditLog.Detail）。
func auditDetail(kv map[string]any) string {
	if len(kv) == 0 {
		return "{}"
	}
	b, err := json.Marshal(kv)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// truncate 按字符数截断，避免超长字段写库失败。
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
