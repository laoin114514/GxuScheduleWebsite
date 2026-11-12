package auth

import (
	"strings"
	"testing"
)

func TestGenerateAndParse(t *testing.T) {
	seen := make(map[string]bool, 20)
	for i := 0; i < 20; i++ {
		plain, keyID, keyHash, last4, err := Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if !strings.HasPrefix(plain, KeyPrefix) {
			t.Fatalf("plaintext %q missing prefix %q", plain, KeyPrefix)
		}
		if got, want := len(plain), len(KeyPrefix)+keyIDLength+1+secretLen; got != want {
			t.Fatalf("plaintext length = %d, want %d", got, want)
		}
		if seen[plain] {
			t.Fatalf("duplicate plaintext generated: %q", plain)
		}
		seen[plain] = true

		gotID, err := ParseKey(plain)
		if err != nil {
			t.Fatalf("ParseKey(%q): %v", plain, err)
		}
		if gotID != keyID {
			t.Fatalf("ParseKey keyID = %q, want %q", gotID, keyID)
		}
		if want := HashKey(plain); keyHash != want {
			t.Fatalf("Generate hash = %q, want %q", keyHash, want)
		}
		if len(keyHash) != 64 {
			t.Fatalf("hash length = %d, want 64", len(keyHash))
		}
		if last4 != plain[len(plain)-4:] {
			t.Fatalf("last4 = %q, want tail of plaintext", last4)
		}
	}
}

func TestParseKeyRejectsMalformed(t *testing.T) {
	valid, _, _, _, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	prefixLen := len(KeyPrefix)
	validID := valid[prefixLen : prefixLen+keyIDLength]
	validSecret := valid[prefixLen+keyIDLength+1:]

	cases := map[string]string{
		"empty":            "",
		"wrong prefix":     "sk_live_" + validID + "_" + validSecret,
		"too short":        KeyPrefix + validID + "_" + validSecret[:len(validSecret)-1],
		"too long":         valid + "x",
		"missing sep":      KeyPrefix + validID + validSecret,
		"bad key id chars": KeyPrefix + "0123456G_" + validSecret,
		"bad secret chars": KeyPrefix + validID + "_" + strings.Repeat("!", secretLen),
		"only prefix":      KeyPrefix,
	}
	for name, raw := range cases {
		if _, err := ParseKey(raw); err == nil {
			t.Fatalf("%s: ParseKey(%q) = nil error, want ErrMalformedKey", name, raw)
		}
	}

	// 前后空白应被容忍（请求头偶发空格）。
	if _, err := ParseKey("  " + valid + "  "); err != nil {
		t.Fatalf("ParseKey with surrounding spaces: %v", err)
	}
}

func TestConstantTimeMatchAny(t *testing.T) {
	keys := []string{"key-one-0123456789", "key-two-0123456789"}
	if !ConstantTimeMatchAny(keys, "key-two-0123456789") {
		t.Fatal("expected match on second candidate")
	}
	if ConstantTimeMatchAny(keys, "key-three-01234567") {
		t.Fatal("unexpected match for unknown key")
	}
	if ConstantTimeMatchAny(keys, "") {
		t.Fatal("empty credential must never match")
	}
	if ConstantTimeMatchAny(nil, "anything") {
		t.Fatal("empty candidate list must never match")
	}
	if ConstantTimeMatchAny([]string{""}, "") {
		t.Fatal("empty candidate entry must never match")
	}
}

func TestLast4(t *testing.T) {
	if got := Last4("gxu_live_abcdef01_1234567890123456789012345678901234567890abc"); got != "0abc" {
		t.Fatalf("Last4 = %q, want 0abc", got)
	}
	if got := Last4("abc"); got != "abc" {
		t.Fatalf("Last4(short) = %q, want abc", got)
	}
}
