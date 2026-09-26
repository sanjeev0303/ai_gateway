package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// apiKeyPrefix marks control-plane API keys. The same prefix is used for
// gateway hot-path keys so both planes share one credential resolver.
const apiKeyPrefix = "aigw_"

// randomHex returns nbytes of cryptographically random bytes hex-encoded.
func randomHex(nbytes int) (string, error) {
	buf := make([]byte, nbytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("identity: generate random token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// normalizeEmail lowercases and trims an email so identity lookups do not
// depend on the case a caller happens to type.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// slugify turns a display name into a URL-safe slug: lowercase alphanumerics
// with every other run collapsed to a single hyphen.
func slugify(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
