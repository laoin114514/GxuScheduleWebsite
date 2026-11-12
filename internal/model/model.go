package model

import "time"

// 应用状态。
const (
	// AppStatusActive 正常：允许上传、latest 正常返回。
	AppStatusActive int8 = 1
	// AppStatusDisabled 停用：拒绝上传、latest 返回 404（官网会回退到内置信息）。
	AppStatusDisabled int8 = 2
)

// 上传 key 状态。
const (
	// UploadKeyStatusActive 当前使用的唯一一把 key（每个 app 同一时刻最多一把）。
	UploadKeyStatusActive int8 = 1
	// UploadKeyStatusRetiring 轮换宽限期：仍可上传，到期后应吊销/自动失效。
	UploadKeyStatusRetiring int8 = 2
	// UploadKeyStatusRevoked 已吊销：立即不可用。
	UploadKeyStatusRevoked int8 = 3
)

// release 状态。
const (
	// ReleaseStatusLive 正常在架，latest 可见。
	ReleaseStatusLive int8 = 1
	// ReleaseStatusYanked 已下架：latest 跳过；已安装用户不受影响（Android 不允许降级）。
	ReleaseStatusYanked int8 = 2
)

// App 应用（渠道）信息表。appKey 全局唯一且永久保留：停用后也不允许别的应用复用。
// 同一个 App 可同时存在多把历史 key 记录，但同一时刻最多一把 status=active，
// 该不变量由 admin 发 key/轮换时在事务里锁 app 行保证。
type App struct {
	ID       uint   `gorm:"primaryKey"`
	AppKey   string `gorm:"size:64;uniqueIndex;not null;comment:应用唯一key（永久保留，不复用）"`
	Name     string `gorm:"size:128;not null;comment:应用显示名"`
	Platform string `gorm:"size:32;not null;default:android;comment:平台"`
	Status   int8   `gorm:"not null;default:1;comment:1 active 2 disabled"`
	// MaxUploadMB 单文件上限；0 表示沿用全局 MAX_UPLOAD_MB。
	MaxUploadMB int64 `gorm:"not null;default:0;comment:0=用全局 MAX_UPLOAD_MB"`
	// DailyUploadLimit 每自然日允许新增的 release 数；幂等重传不计数。
	DailyUploadLimit int  `gorm:"not null;default:20;comment:每自然日非幂等上传次数上限"`
	AllowDowngrade   bool `gorm:"not null;default:false;comment:允许上传低于当前最新 versionCode 的包"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AppUploadKey 每应用上传密钥。明文只在创建/轮换响应里返回一次，
// 库里只存 SHA-256；keyID（8 位 hex）是公开标识，用于查库、审计与展示。
type AppUploadKey struct {
	ID      uint   `gorm:"primaryKey"`
	AppID   uint   `gorm:"not null;index;comment:apps.id"`
	KeyID   string `gorm:"size:8;uniqueIndex;not null;comment:明文 key 中的公开标识"`
	KeyHash string `gorm:"size:64;uniqueIndex;not null;comment:SHA256(完整明文 key)"`
	Last4   string `gorm:"size:4;not null;comment:仅用于展示，不泄露可用信息"`
	Label   string `gorm:"size:64;not null;comment:用途标签，如 gha-release / local-beta"`
	Status  int8   `gorm:"not null;default:1;comment:1 active 2 retiring 3 revoked"`
	// ExpiresAt 为空表示不过期。retiring 宽限期也靠它兜底。
	ExpiresAt    *time.Time
	LastUsedAt   *time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
	RevokeReason string `gorm:"size:255"`
}

// Release 上传记录。只存 OSS 对象 key,完整下载地址由响应时拼接/签名生成。
// 唯一约束 (app_key, version_code) 防止同版本重复插入；
// status=2 表示已下架，latest 会跳过它（不删除记录与 OSS 对象，便于恢复/审计）。
type Release struct {
	ID          uint   `gorm:"primaryKey"`
	AppKey      string `gorm:"size:64;not null;uniqueIndex:uk_app_version,priority:1;index:idx_app_status_version,priority:1"`
	VersionCode int64  `gorm:"not null;uniqueIndex:uk_app_version,priority:2;index:idx_app_status_version,priority:3"`
	VersionName string `gorm:"size:64;not null"`
	FileName    string `gorm:"size:256;not null;comment:原始文件名"`
	FileKey     string `gorm:"size:512;not null;comment:OSS对象key，如release/{appKey}/{versionCode}/{file}"`
	SHA256      string `gorm:"size:64;not null"`
	Size        int64  `gorm:"not null"`
	Changelog   string `gorm:"type:text"`
	Forced      bool   `gorm:"not null;default:false;comment:强制更新"`
	Status      int8   `gorm:"not null;default:1;index:idx_app_status_version,priority:2;comment:1 live 2 yanked"`
	// UploadKeyID 本次上传所用 app_upload_keys.id；legacy 全局 key 为 0。
	UploadKeyID uint `gorm:"not null;default:0;comment:上传所用 app_upload_keys.id；legacy key 为 0"`
	YankedAt    *time.Time
	YankedBy    string `gorm:"size:64;comment:下架操作者（admin）"`
	YankReason  string `gorm:"size:255"`
	CreatedAt   time.Time
}

// AuditLog 操作审计。只记录元信息，绝不落明文 key。
type AuditLog struct {
	ID         uint   `gorm:"primaryKey"`
	RequestID  string `gorm:"size:32;index;comment:请求 ID，便于串联日志"`
	ActorType  string `gorm:"size:16;not null;comment:admin/app/legacy/anonymous"`
	ActorKeyID string `gorm:"size:8"`
	Action     string `gorm:"size:64;not null;index;comment:upload/app.create/key.issue/..."`
	AppKey     string `gorm:"size:64;index"`
	TargetType string `gorm:"size:32"`
	TargetID   string `gorm:"size:64"`
	IP         string `gorm:"size:64"`
	UserAgent  string `gorm:"size:255"`
	// Detail JSON 文本，如 {"reason":"...","versionCode":123}。
	Detail    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}
