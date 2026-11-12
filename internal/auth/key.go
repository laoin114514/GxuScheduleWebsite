// Package auth 负责每应用上传密钥的生成、解析、哈希，以及把请求头里的
// 明文 key 解析成 app + key 记录（Resolver 接口，DB 实现见 resolver.go）。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// KeyPrefix 上传 key 前缀。`live` 表示生产环境，前缀便于 gitleaks 等扫描器识别泄露。
	KeyPrefix = "gxu_live_"

	keyIDLength = 8  // 4 字节 hex，公开标识，用于查库/审计，不是机密
	secretBytes = 32 // 256 bit 随机 secret
	secretLen   = 43 // base64url(32B) 长度固定 43
)

var (
	// ErrMalformedKey key 格式不合法（不对外区分具体原因，避免侧信道）。
	ErrMalformedKey = errors.New("malformed upload key")

	keyIDPattern  = regexp.MustCompile(`^[0-9a-f]{8}$`)
	secretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
)

// Generate 生成一把新上传 key。
// 返回：明文（只应返回给调用方一次）、公开 keyID、入库用 SHA-256、展示用 last4。
func Generate() (plaintext, keyID, keyHash, last4 string, err error) {
	idBytes := make([]byte, keyIDLength/2)
	if _, err = rand.Read(idBytes); err != nil {
		return "", "", "", "", fmt.Errorf("generate key id: %w", err)
	}
	secret := make([]byte, secretBytes)
	if _, err = rand.Read(secret); err != nil {
		return "", "", "", "", fmt.Errorf("generate key secret: %w", err)
	}
	keyID = hex.EncodeToString(idBytes)
	secretText := base64.RawURLEncoding.EncodeToString(secret)
	plaintext = KeyPrefix + keyID + "_" + secretText
	return plaintext, keyID, HashKey(plaintext), Last4(plaintext), nil
}

// ParseKey 从明文 key 中取出公开 keyID，并校验整体格式。
// 注意 secret 使用 base64url，可能包含 '_'，因此按固定长度切分而不是按分隔符 split。
func ParseKey(raw string) (keyID string, err error) {
	raw = strings.TrimSpace(raw)
	const prefixLen = len(KeyPrefix)
	if len(raw) != prefixLen+keyIDLength+1+secretLen || !strings.HasPrefix(raw, KeyPrefix) {
		return "", ErrMalformedKey
	}
	keyID = raw[prefixLen : prefixLen+keyIDLength]
	if !keyIDPattern.MatchString(keyID) || raw[prefixLen+keyIDLength] != '_' {
		return "", ErrMalformedKey
	}
	if !secretPattern.MatchString(raw[prefixLen+keyIDLength+1:]) {
		return "", ErrMalformedKey
	}
	return keyID, nil
}

// HashKey 返回完整明文 key 的 SHA-256（hex）。
// 256 bit 高熵随机串不需要 bcrypt/Argon2：慢哈希还会让按 keyID 查库后的比对变慢，
// 这里只要求「库泄露后无法反推明文 + 常量时间比较」。
func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Last4 返回明文最后 4 位，仅用于列表/日志展示（配合 keyID 足够定位，不足以还原 key）。
func Last4(raw string) string {
	if len(raw) <= 4 {
		return raw
	}
	return raw[len(raw)-4:]
}

// ConstantTimeMatchAny 常量时间比较 got 是否等于 candidates 中任意一个。
// 空列表或空串一律返回 false（避免「空配置匹配空请求头」）。
func ConstantTimeMatchAny(candidates []string, got string) bool {
	if got == "" {
		return false
	}
	matched := 0
	for _, want := range candidates {
		if want == "" {
			continue
		}
		matched |= subtle.ConstantTimeCompare([]byte(got), []byte(want))
	}
	return matched == 1
}
