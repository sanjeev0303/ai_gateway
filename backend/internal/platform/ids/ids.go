// Package ids generates collision-resistant identifiers from crypto/rand.
// Identifiers are opaque 128-bit values (32 lowercase hex chars); nothing in
// the system should ever parse meaning out of them.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// New returns a 128-bit random identifier as 32 lowercase hex characters.
func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read only fails on supported platforms when the
		// entropy source is unavailable; minting predictable IDs then would
		// be a security hole, so failing fast is the only safe option.
		panic(fmt.Sprintf("ids: crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

// NewCorrelationID is New() under a self-documenting name for correlation
// tracking across services.
func NewCorrelationID() string { return New() }
