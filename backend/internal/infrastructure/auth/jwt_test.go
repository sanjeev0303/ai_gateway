package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/auth"
	"ai-governance-saas/backend/internal/ports"
)

const testIssuer = "test-issuer"

// testKey returns a fixed 32-byte key so the suite is deterministic.
func testKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func mustManager(t *testing.T, ttl time.Duration, key []byte) (ports.TokenIssuer, ports.TokenVerifier) {
	t.Helper()
	issuer, verifier, err := auth.NewHMACTokenManager(testIssuer, ttl, key)
	if err != nil {
		t.Fatalf("NewHMACTokenManager: %v", err)
	}
	return issuer, verifier
}

func fullClaims() domain.TokenClaims {
	return domain.TokenClaims{
		Subject:        uuid.NewString(),
		Email:          "alice@example.test",
		DisplayName:    "Alice",
		OrganizationID: uuid.New(),
		Role:           domain.RoleAdmin,
		Kind:           domain.TokenKindUser,
	}
}

func TestNewHMACTokenManagerValidation(t *testing.T) {
	tests := []struct {
		name         string
		issuer       string
		ttl          time.Duration
		key          []byte
		wantErr      bool
		wantSentinel error
	}{
		{name: "empty issuer", issuer: "", ttl: time.Hour, key: testKey(), wantErr: true},
		{name: "zero ttl", issuer: testIssuer, ttl: 0, key: testKey(), wantErr: true},
		{name: "negative ttl", issuer: testIssuer, ttl: -time.Minute, key: testKey(), wantErr: true},
		{name: "key 16 bytes", issuer: testIssuer, ttl: time.Hour, key: []byte("0123456789abcdef"), wantErr: true, wantSentinel: auth.ErrWeakSigningKey},
		{name: "key 31 bytes", issuer: testIssuer, ttl: time.Hour, key: []byte("0123456789abcdef0123456789abcd"), wantErr: true, wantSentinel: auth.ErrWeakSigningKey},
		{name: "key exactly 32 bytes", issuer: testIssuer, ttl: time.Hour, key: testKey()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := auth.NewHMACTokenManager(tt.issuer, tt.ttl, tt.key)
			if tt.wantSentinel != nil {
				if !errors.Is(err, tt.wantSentinel) {
					t.Fatalf("expected %v, got %v", tt.wantSentinel, err)
				}
				return
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewHMACTokenManagerValidKey(t *testing.T) {
	issuer, verifier, err := auth.NewHMACTokenManager(testIssuer, time.Hour, testKey())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if issuer == nil || verifier == nil {
		t.Fatal("expected non-nil issuer and verifier")
	}
}

func TestIssueVerifyRoundTrip(t *testing.T) {
	issuer, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()
	want := fullClaims()

	raw, err := issuer.Issue(ctx, want)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := verifier.Verify(ctx, raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Subject != want.Subject ||
		got.Email != want.Email ||
		got.DisplayName != want.DisplayName ||
		got.OrganizationID != want.OrganizationID ||
		got.Role != want.Role ||
		got.Kind != want.Kind {
		t.Fatalf("claims mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestIssueDefaultsTimesToTTL(t *testing.T) {
	issuer, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	claims := fullClaims()
	claims.IssuedAt = time.Time{}
	claims.ExpiresAt = time.Time{}

	raw, err := issuer.Issue(ctx, claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := verifier.Verify(ctx, raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.IssuedAt.IsZero() || got.ExpiresAt.IsZero() {
		t.Fatal("expected IssuedAt and ExpiresAt to be set")
	}
	if got.ExpiresAt.Before(got.IssuedAt) {
		t.Fatalf("expires %v before issued %v", got.ExpiresAt, got.IssuedAt)
	}
}

func TestIssueNilOrgIDRoundTrip(t *testing.T) {
	issuer, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	claims := fullClaims()
	claims.OrganizationID = uuid.Nil

	raw, err := issuer.Issue(ctx, claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := verifier.Verify(ctx, raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.OrganizationID != uuid.Nil {
		t.Fatalf("expected nil org id, got %v", got.OrganizationID)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	issuer, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	claims := fullClaims()
	claims.ExpiresAt = time.Now().UTC().Add(-time.Minute)

	raw, err := issuer.Issue(ctx, claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	_, err = verifier.Verify(ctx, raw)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	issuer, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	raw, err := issuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("malformed JWT shape: %d parts", len(parts))
	}
	claimsPart := []byte(parts[1])
	claimsPart[0] ^= 0x01
	parts[1] = string(claimsPart)
	tampered := strings.Join(parts, ".")

	_, err = verifier.Verify(ctx, tampered)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for tampered claims, got %v", err)
	}

	parts[1] = strings.Split(raw, ".")[1]
	sig := []byte(parts[2])
	sig[0] ^= 0x01
	parts[2] = string(sig)
	_, err = verifier.Verify(ctx, strings.Join(parts, "."))
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for tampered signature, got %v", err)
	}
}

func TestVerifyRejectsForeignIssuer(t *testing.T) {
	_, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	foreignIssuer, _, err := auth.NewHMACTokenManager("other-issuer", time.Hour, testKey())
	if err != nil {
		t.Fatalf("NewHMACTokenManager: %v", err)
	}
	raw, err := foreignIssuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	_, err = verifier.Verify(ctx, raw)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for foreign issuer, got %v", err)
	}
}

func TestVerifyRejectsForeignKey(t *testing.T) {
	issuer, _ := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	_, foreignVerifier, err := auth.NewHMACTokenManager(testIssuer, time.Hour, []byte("fedcba9876543210fedcba9876543210"))
	if err != nil {
		t.Fatalf("NewHMACTokenManager: %v", err)
	}
	raw, err := issuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	_, err = foreignVerifier.Verify(ctx, raw)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for foreign key, got %v", err)
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	_, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	for _, raw := range []string{"", "not a jwt", "aaa.bbb.ccc"} {
		if _, err := verifier.Verify(ctx, raw); !errors.Is(err, auth.ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken for %q, got %v", raw, err)
		}
	}
}

func TestVerifyRejectsAlgNone(t *testing.T) {
	_, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"attacker"}`))
	raw := header + "." + payload + "."

	_, err := verifier.Verify(ctx, raw)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for alg none, got %v", err)
	}
}

func TestVerifyRequiresExpiration(t *testing.T) {
	_, verifier := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "sub",
		Issuer:  testIssuer,
	})
	raw, err := unsigned.SignedString(testKey())
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	_, err = verifier.Verify(ctx, raw)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for missing exp, got %v", err)
	}
}

func mustRotatingManager(t *testing.T, ttl time.Duration, key []byte, previousKeys ...[]byte) (ports.TokenIssuer, ports.TokenVerifier) {
	t.Helper()
	issuer, verifier, err := auth.NewRotatingHMACTokenManager(testIssuer, ttl, key, previousKeys...)
	if err != nil {
		t.Fatalf("NewRotatingHMACTokenManager: %v", err)
	}
	return issuer, verifier
}

func TestIssueStampsKID(t *testing.T) {
	issuer, _ := mustManager(t, time.Hour, testKey())
	ctx := context.Background()

	raw, err := issuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	sum := sha256.Sum256(testKey())
	if kid := parsed.Header["kid"]; kid != hex.EncodeToString(sum[:8]) {
		t.Fatalf("kid mismatch: got %v, want %s", kid, hex.EncodeToString(sum[:8]))
	}
}

func TestRotatingManagerAcceptsPreviousKeys(t *testing.T) {
	ctx := context.Background()
	previousKey := []byte("fedcba9876543210fedcba9876543210")

	// A token issued while previousKey was current must still verify once the
	// key has rotated into the previous slot of the ring.
	previousIssuer, _ := mustManager(t, time.Hour, previousKey)
	raw, err := previousIssuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, verifier := mustRotatingManager(t, time.Hour, testKey(), previousKey)
	got, err := verifier.Verify(ctx, raw)
	if err != nil || got.Subject == "" {
		t.Fatalf("token signed with previous key must verify (claims=%+v, err=%v)", got, err)
	}
}

func TestRotatingManagerRejectsUnrelatedKey(t *testing.T) {
	ctx := context.Background()
	issuer, _ := mustManager(t, time.Hour, testKey())
	raw, err := issuer.Issue(ctx, fullClaims())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, verifier := mustRotatingManager(t, time.Hour, []byte("abcdef0123456789abcdef0123456789"))
	if _, err := verifier.Verify(ctx, raw); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for unknown kid, got %v", err)
	}
}

func TestVerifyRejectsForgedKID(t *testing.T) {
	ctx := context.Background()
	_, verifier := mustRotatingManager(t, time.Hour, testKey(), []byte("fedcba9876543210fedcba9876543210"))

	// Signed with a real ring key, but the header names a kid that is not in
	// the ring: verification must reject instead of ignoring the header.
	forged := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "sub",
		Issuer:    testIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
	})
	forged.Header["kid"] = "0000000000000000"
	raw, err := forged.SignedString(testKey())
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	if _, err := verifier.Verify(ctx, raw); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for forged kid, got %v", err)
	}
}

func TestVerifyLegacyTokenWithoutKID(t *testing.T) {
	ctx := context.Background()

	// Tokens minted before kid stamping carry no kid header; they must still
	// verify against a ring key (probed current-first).
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "sub",
		Issuer:    testIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
	})
	previousKey := []byte("fedcba9876543210fedcba9876543210")
	raw, err := legacy.SignedString(previousKey)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, verifier := mustRotatingManager(t, time.Hour, testKey(), previousKey)
	if _, err := verifier.Verify(ctx, raw); err != nil {
		t.Fatalf("headerless legacy token must verify, got %v", err)
	}
}

func TestRotatingManagerRejectsWeakPreviousKey(t *testing.T) {
	if _, _, err := auth.NewRotatingHMACTokenManager(testIssuer, time.Hour, testKey(), []byte("too-short")); !errors.Is(err, auth.ErrWeakSigningKey) {
		t.Fatalf("expected ErrWeakSigningKey for weak previous key, got %v", err)
	}
}

func TestSHA256TokenHasher(t *testing.T) {
	hasher := auth.NewSHA256TokenHasher()
	ctx := context.Background()

	first, err := hasher.Hash(ctx, "secret-1")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	second, err := hasher.Hash(ctx, "secret-2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if first == second {
		t.Fatal("distinct secrets must produce distinct digests")
	}

	again, err := hasher.Hash(ctx, "secret-1")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if again != first {
		t.Fatalf("hashing is not deterministic: %q vs %q", again, first)
	}

	if !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("missing sha256: prefix: %q", first)
	}
	sum := sha256.Sum256([]byte("secret-1"))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if first != want {
		t.Fatalf("digest mismatch:\n got %q\nwant %q", first, want)
	}
}
