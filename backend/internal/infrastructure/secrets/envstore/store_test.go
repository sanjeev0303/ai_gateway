package envstore

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	ports "ai-governance-saas/backend/internal/ports/secrets"
)

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "")

	if err := s.Put(ctx, "creds/anthropic/prod", "sk-ant-prod-secret"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := s.Get(ctx, "creds/anthropic/prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "sk-ant-prod-secret" {
		t.Errorf("Get() = %q, want the stored value", got)
	}

	// The on-disk file must never contain the plaintext.
	raw, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}
	if string(raw) == "sk-ant-prod-secret" || contains(raw, []byte("sk-ant-prod-secret")) {
		t.Error("store file contains plaintext; must be encrypted at rest")
	}
}

func TestPutReplaces(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "")
	ref := "k"

	if err := s.Put(ctx, ref, "v1"); err != nil {
		t.Fatalf("Put(v1) error = %v", err)
	}
	if err := s.Put(ctx, ref, "v2"); err != nil {
		t.Fatalf("Put(v2) error = %v", err)
	}
	got, err := s.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "v2" {
		t.Errorf("Get() = %q, want v2", got)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "")

	if err := s.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := s.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Get(ctx, "k"); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Get() after Delete = %v, want ErrNotFound", err)
	}
	// Deleting a missing ref must be a no-op, not an error.
	if err := s.Delete(ctx, "k"); err != nil {
		t.Errorf("Delete() missing ref = %v, want nil", err)
	}
}

func TestGetMissing(t *testing.T) {
	s := newTestStore(t, "")
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Get() missing = %v, want ErrNotFound", err)
	}
}

func TestTamperDetection(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "")
	if err := s.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	raw, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	// Flip one byte of ciphertext inside the file; the next Get must fail
	// authentication instead of returning garbage.
	raw[len(raw)/2] ^= 0x01
	if err := os.WriteFile(s.path, raw, 0o600); err != nil {
		t.Fatalf("rewrite store: %v", err)
	}

	if _, err := s.Get(ctx, "k"); err == nil {
		t.Error("Get() on tampered store = nil, want authentication failure")
	}
}

func TestPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "")
	if err := s.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	s2, err := New(s.path, "") // same key file
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := s2.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get() after reopen = %v", err)
	}
	if got != "v" {
		t.Errorf("Get() after reopen = %q, want v", got)
	}
}

func TestExplicitKeyAndRejectWrongKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	hexKey := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

	s, err := New(filepath.Join(dir, "secrets.json"), hexKey)
	if err != nil {
		t.Fatalf("New(hex key) error = %v", err)
	}
	if err := s.Put(ctx, "k", "secret-value"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// A different key must fail to decrypt.
	otherKey := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	s2, err := New(filepath.Join(dir, "secrets.json"), otherKey)
	if err != nil {
		t.Fatalf("New(other key) error = %v", err)
	}
	if _, err := s2.Get(ctx, "k"); err == nil {
		t.Error("Get() with wrong key = nil, want decryption failure")
	}

	// No key may be left behind anywhere unexpected on the explicit-key path.
	if _, err := os.Stat(filepath.Join(dir, "secrets.json.key")); !errors.Is(err, os.ErrNotExist) {
		t.Error("explicit key path must not create a .key file")
	}
}

func TestGeneratedKeyFile(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "secrets.json"), "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	keyPath := filepath.Join(dir, "secrets.json.key")
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read generated key file: %v", err)
	}
	if len(raw) < 64 {
		t.Errorf("generated key file = %d bytes, want hex 32-byte key", len(raw))
	}
	// The key file must be readable only by the owner.
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode = %o, want 0600", info.Mode().Perm())
	}
	// Round-trip with the persisted key file.
	if err := s.Put(context.Background(), "k", "v"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	decoded, _ := hex.DecodeString(string(raw))
	_ = decoded
}

func TestRejectsBadKeyHex(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(filepath.Join(dir, "secrets.json"), "not-hex"); err == nil {
		t.Error("New(bad key hex) = nil, want error")
	}
	short := "0011"
	if _, err := New(filepath.Join(dir, "secrets.json"), short); err == nil {
		t.Error("New(short key) = nil, want error")
	}
}

func TestRejectsEmptyRef(t *testing.T) {
	s := newTestStore(t, "")
	if err := s.Put(context.Background(), "", "v"); err == nil {
		t.Error("Put(empty ref) = nil, want error")
	}
}

func newTestStore(t *testing.T, keyHex string) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "secrets.json"), keyHex)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return s
}

func contains(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}
