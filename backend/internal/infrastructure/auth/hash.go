package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"ai-governance-saas/backend/internal/ports"
)

// sha256Hasher digests credential secrets with sha256.
type sha256Hasher struct{}

// NewSHA256TokenHasher returns a TokenHasher producing "sha256:<hex>" digests.
func NewSHA256TokenHasher() ports.TokenHasher {
	return sha256Hasher{}
}

func (sha256Hasher) Hash(_ context.Context, secret string) (string, error) {
	sum := sha256.Sum256([]byte(secret))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
