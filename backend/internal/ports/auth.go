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
