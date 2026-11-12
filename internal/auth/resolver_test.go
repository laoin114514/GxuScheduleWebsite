package auth

import (
	"context"
	"testing"
)

func TestDBResolverNilDBIsSafe(t *testing.T) {
	resolver := NewDBResolver(nil)
	app, key, err := resolver.ResolveUploadKey(context.Background(), "anything")
	if err == nil {
		t.Fatal("nil db resolver should return error")
	}
	if err != ErrKeyNotFound {
		t.Fatalf("err = %v, want ErrKeyNotFound", err)
	}
	if app != nil || key != nil {
		t.Fatalf("app/key = %v/%v, want nil", app, key)
	}
	// TouchKeyLastUsed 在 nil db 下也必须安全。
	resolver.TouchKeyLastUsed(context.Background(), nil)
}

func TestDBResolverMalformedKey(t *testing.T) {
	resolver := NewDBResolver(nil)
	if _, _, err := resolver.ResolveUploadKey(context.Background(), "not-a-valid-key"); err != ErrKeyNotFound {
		t.Fatalf("err = %v, want ErrKeyNotFound", err)
	}
}
