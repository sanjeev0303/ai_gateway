// Package auth implements the token machinery behind the authentication
// ports: HS256-signed JWTs (TokenIssuer/TokenVerifier) plus one-way secret
// hashing for invite tokens and API keys (TokenHasher).
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
)

var (
	// ErrInvalidToken wraps every rejection a Verify returns: bad signature,
	// expiry, wrong issuer, unknown key id, or malformed claims.
	ErrInvalidToken = errors.New("auth: invalid token")
	// ErrWeakSigningKey guards the 32-byte minimum for HS256.
	ErrWeakSigningKey = errors.New("auth: HMAC signing key must be at least 32 bytes")
)

const minSigningKeyBytes = 32

// jwtClaims is the wire shape of domain.TokenClaims.
type jwtClaims struct {
	Email          string           `json:"email,omitempty"`
	DisplayName    string           `json:"display_name,omitempty"`
	OrganizationID string           `json:"org_id,omitempty"`
	Role           string           `json:"role,omitempty"`
	Kind           domain.TokenKind `json:"kind"`
	jwt.RegisteredClaims
}

// hmacJWT signs and verifies HS256 JWTs scoped to one issuer.
//
// Key rotation: Issue always signs with the current key and stamps a kid
// (key id) header so a verifying process can select the matching key. Verify
// accepts the current key plus any configured previous keys (a small ring,
// typically current + 2 previous). Tokens issued before kid stamping carried
// no header; they still verify as long as they validate against one of the
// ring keys, which keeps in-flight tokens alive through a deployment.
type hmacJWT struct {
	issuer     string
	ttl        time.Duration
	currentKey []byte
	currentKID string
	// ring maps a kid to its key material; order lists the kids current-first
	// so a headerless legacy token is tried against the newest key first.
	ring  map[string][]byte
	order []string
}

// NewHMACTokenManager returns the TokenIssuer and TokenVerifier for one HS256
// key. The key must be at least 32 random bytes; reuse the same key across
// processes so tokens issued by one instance verify on all others.
func NewHMACTokenManager(issuer string, ttl time.Duration, signingKey []byte) (ports.TokenIssuer, ports.TokenVerifier, error) {
	return NewRotatingHMACTokenManager(issuer, ttl, signingKey)
}

// NewRotatingHMACTokenManager returns a TokenIssuer/TokenVerifier backed by a
// key ring: signingKey is always used to issue, and verification additionally
// accepts previousKeys while they rotate out of service. Every key must be at
// least 32 random bytes. Use the same set of keys across processes so tokens
// issued by one instance verify on all others.
func NewRotatingHMACTokenManager(issuer string, ttl time.Duration, signingKey []byte, previousKeys ...[]byte) (ports.TokenIssuer, ports.TokenVerifier, error) {
	if issuer == "" || ttl <= 0 {
		return nil, nil, errors.New("auth: issuer and ttl must be set")
	}
	if len(signingKey) < minSigningKeyBytes {
		return nil, nil, ErrWeakSigningKey
	}
	for _, pk := range previousKeys {
		if len(pk) < minSigningKeyBytes {
			return nil, nil, ErrWeakSigningKey
		}
	}

	ring := make(map[string][]byte, len(previousKeys)+1)
	order := make([]string, 0, len(previousKeys)+1)

	currentKID := kidOf(signingKey)
	ring[currentKID] = signingKey
	order = append(order, currentKID)
	for _, pk := range previousKeys {
		k := kidOf(pk)
		if _, ok := ring[k]; ok {
			continue // duplicate key material: keep the first occurrence
		}
		ring[k] = pk
		order = append(order, k)
	}

	m := &hmacJWT{
		issuer:     issuer,
		ttl:        ttl,
		currentKey: signingKey,
		currentKID: currentKID,
		ring:       ring,
		order:      order,
	}
	return m, m, nil
}

// kidOf content-addresses a key: identical key material always yields the same
// kid, so every process derives the same ring without shared rotation state.
func kidOf(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:8])
}

func (m *hmacJWT) Issue(_ context.Context, claims domain.TokenClaims) (string, error) {
	now := time.Now().UTC()
	if claims.IssuedAt.IsZero() {
		claims.IssuedAt = now
	}
	if claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = now.Add(m.ttl)
	}
	var orgID string
	if claims.OrganizationID != uuid.Nil {
		orgID = claims.OrganizationID.String()
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		Email:          claims.Email,
		DisplayName:    claims.DisplayName,
		OrganizationID: orgID,
		Role:           claims.Role,
		Kind:           claims.Kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.Subject,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(claims.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
		},
	})
	// Stamp the key id so verifiers pick the same key out of their ring.
	tok.Header["kid"] = m.currentKID
	raw, err := tok.SignedString(m.currentKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return raw, nil
}

func (m *hmacJWT) Verify(_ context.Context, raw string) (domain.TokenClaims, error) {
	// Read the kid from the unverified header so the ring can select the
	// candidate key(s); the signature itself is never trusted here.
	parsed, _, err := jwt.NewParser().ParseUnverified(raw, &jwtClaims{})
	if err != nil {
		return domain.TokenClaims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	candidates := m.order // headerless legacy token: try every ring key
	if kid, ok := parsed.Header["kid"].(string); ok && kid != "" {
		if _, known := m.ring[kid]; !known {
			return domain.TokenClaims{}, fmt.Errorf("%w: unknown key id", ErrInvalidToken)
		}
		candidates = []string{kid}
	}

	var lastErr error
	for _, kid := range candidates {
		claims := &jwtClaims{}
		validated, err := jwt.ParseWithClaims(raw, claims,
			func(t *jwt.Token) (any, error) { return m.ring[kid], nil },
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(m.issuer),
			jwt.WithExpirationRequired(),
		)
		if err == nil && validated.Valid {
			orgID, perr := uuid.Parse(claims.OrganizationID)
			if perr != nil {
				orgID = uuid.Nil
			}
			// Legacy tokens predating kid stamping may lack iat; the expiry
			// check above requires exp, but iat is optional.
			var issuedAt, expiresAt time.Time
			if claims.IssuedAt != nil {
				issuedAt = claims.IssuedAt.Time
			}
			if claims.ExpiresAt != nil {
				expiresAt = claims.ExpiresAt.Time
			}
			return domain.TokenClaims{
				Subject:        claims.Subject,
				Email:          claims.Email,
				DisplayName:    claims.DisplayName,
				OrganizationID: orgID,
				Role:           claims.Role,
				Kind:           claims.Kind,
				IssuedAt:       issuedAt,
				ExpiresAt:      expiresAt,
			}, nil
		}
		lastErr = err
	}
	return domain.TokenClaims{}, fmt.Errorf("%w: %v", ErrInvalidToken, lastErr)
}