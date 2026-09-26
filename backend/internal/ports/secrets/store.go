// Package secrets defines the SecretStore port (M4). The gateway and control
// plane hold provider API keys by opaque reference; the concrete store (local
// encrypted file, Vault, …) is chosen at bootstrap and swapped without
// touching the business layer.
package secrets

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a reference has no stored value.
var ErrNotFound = errors.New("secrets: not found")

// Store persists provider secrets at rest and retrieves them by opaque
// reference. Implementations must encrypt at rest, must never include the
// value in error messages, and must never log values.
type Store interface {
	// Get returns the stored value for ref, or ErrNotFound.
	Get(ctx context.Context, ref string) (string, error)
	// Put stores value under ref, replacing any previous value.
	Put(ctx context.Context, ref, value string) error
	// Delete removes ref. Deleting a missing ref is a no-op.
	Delete(ctx context.Context, ref string) error
}