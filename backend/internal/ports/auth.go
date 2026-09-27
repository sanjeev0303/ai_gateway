// Package ports defines the application-layer ports (interfaces) that control
// the boundaries of the AI Governance backend. Repository ports live in
// repositories/; this file holds non-persistence ports such as token signing.
package ports

import (
	"context"

	"ai-governance-saas/backend/internal/domain"
)

// TokenIssuer mints signed JWTs for authenticated principals.
type TokenIssuer interface {
	Issue(ctx context.Context, claims domain.TokenClaims) (string, error)
}

// TokenVerifier validates a signed JWT and returns its claims. Verify
// rejects expired, tampered, or foreign-issuer tokens.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (domain.TokenClaims, error)
}

// TokenHasher produces the one-way digest of a credential secret before it is
// persisted (invite tokens, API keys). The raw secret is returned to the
// caller once and never stored.
type TokenHasher interface {
	Hash(ctx context.Context, secret string) (string, error)
}

// PasswordHasher derives a storable digest from a user-chosen password and
// verifies a candidate against it.
//
// This is deliberately NOT the same contract as TokenHasher. TokenHasher may
// digest with plain SHA-256 because its inputs are high-entropy random
// secrets. A user-chosen password is low-entropy, so a fast digest is
// reversible at scale by brute force; the implementation must be a slow,
// salted KDF.
type PasswordHasher interface {
	// Hash returns a self-describing encoded digest that is safe to persist.
	// The encoding carries its own algorithm and cost parameters so they can
	// be raised later without a schema change or a flag day.
	Hash(ctx context.Context, password string) (string, error)
	// Verify reports whether password matches the encoded digest. It must
	// return one uniform error for a mismatch, an unknown email, and
	// unparsable stored input, so callers cannot distinguish them.
	Verify(ctx context.Context, encoded, password string) error
	// DecoyDigest returns a well-formed digest of a secret nobody holds.
	// Callers verify against it when there is no stored digest to check --
	// an unknown email, or an account that has never set a password -- so
	// those paths cost the same as a real one and cannot be told apart by
	// response time.
	DecoyDigest() string
}
