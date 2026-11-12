package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/laoin114514/gxuschedule-server/internal/model"
)

// Resolver 把上传请求里的明文 key 解析成 app + key 记录。
// 抽成接口是为了让 handler/middleware 可以注入 fake 做单测。
var (
	// ErrKeyNotFound key 不存在/格式不对/哈希不匹配（对外统一按 401 处理，不区分细节）。
	ErrKeyNotFound = errors.New("upload key not found")
	// ErrKeyRevoked key 已被吊销。
	ErrKeyRevoked = errors.New("upload key revoked")
	// ErrKeyExpired key 已过期（retiring 宽限期结束等）。
	ErrKeyExpired = errors.New("upload key expired")
	// ErrAppDisabled app 已被停用。
	ErrAppDisabled = errors.New("app disabled")
)

type Resolver interface {
	// ResolveUploadKey 解析失败时返回上述 sentinel error（或包装后的 DB 错误）。
	ResolveUploadKey(ctx context.Context, raw string) (*model.App, *model.AppUploadKey, error)
	// TouchKeyLastUsed 尽力更新 last_used_at（内部做每分钟节流），失败不影响请求。
	TouchKeyLastUsed(ctx context.Context, key *model.AppUploadKey)
}

// DBResolver 基于 MySQL 的实现。
type DBResolver struct {
	db *gorm.DB
}

func NewDBResolver(db *gorm.DB) *DBResolver {
	return &DBResolver{db: db}
}

func (r *DBResolver) ResolveUploadKey(ctx context.Context, raw string) (*model.App, *model.AppUploadKey, error) {
	// db 为 nil 只会出现在测试里：按 key 不存在处理，保证鉴权失败路径不 panic。
	if r.db == nil {
		return nil, nil, ErrKeyNotFound
	}
	keyID, err := ParseKey(raw)
	if err != nil {
		return nil, nil, ErrKeyNotFound
	}

	var key model.AppUploadKey
	if err := r.db.WithContext(ctx).Where("key_id = ?", keyID).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrKeyNotFound
		}
		return nil, nil, fmt.Errorf("query upload key: %w", err)
	}
	// key_id 是公开的，必须再校验完整哈希，防止伪造 keyID 撞库。
	if !ConstantTimeMatchAny([]string{key.KeyHash}, HashKey(raw)) {
		return nil, nil, ErrKeyNotFound
	}

	switch key.Status {
	case model.UploadKeyStatusActive, model.UploadKeyStatusRetiring:
		// retiring 是轮换宽限期，仍允许上传。
	case model.UploadKeyStatusRevoked:
		return nil, nil, ErrKeyRevoked
	default:
		return nil, nil, ErrKeyNotFound
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, nil, ErrKeyExpired
	}

	var app model.App
	if err := r.db.WithContext(ctx).Where("id = ?", key.AppID).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrKeyNotFound
		}
		return nil, nil, fmt.Errorf("query app: %w", err)
	}
	if app.Status != model.AppStatusActive {
		return &app, &key, ErrAppDisabled
	}
	return &app, &key, nil
}

// TouchKeyLastUsed 更新 last_used_at；每分钟最多写一次，避免每个请求都写库。
func (r *DBResolver) TouchKeyLastUsed(ctx context.Context, key *model.AppUploadKey) {
	if r.db == nil || key == nil {
		return
	}
	now := time.Now()
	if key.LastUsedAt != nil && now.Sub(*key.LastUsedAt) < time.Minute {
		return
	}
	if err := r.db.WithContext(ctx).Model(&model.AppUploadKey{}).
		Where("id = ?", key.ID).Update("last_used_at", now).Error; err == nil {
		key.LastUsedAt = &now
	}
}
