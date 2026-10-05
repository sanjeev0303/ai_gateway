// Package repositories defines the persistence ports (interfaces) that the
// application layer depends on. Implementations live in
// internal/infrastructure/postgres/repositories and translate between these
// contracts and the sqlc-generated DTOs.
//
// Tenancy is explicit: tenant-scoped methods take the organization ID as a
// parameter rather than hiding it in the receiver or context. The database
// still enforces it via RLS within the caller's tenant context.
//
// Get-style methods return ErrNotFound when the resource does not exist, and
// idempotent Record/Create methods return ErrConflict on a duplicate key.
package repositories

import "errors"

// Sentinel errors returned by repository implementations.
var (
	// ErrNotFound is returned when a queried resource does not exist.
	ErrNotFound = errors.New("repositories: not found")
	// ErrConflict is returned when an idempotent insert collides with an
	// existing row (duplicate request, idempotency key, or billing event).
	// Callers may retry the operation (e.g. org provisioning under a new slug).
	ErrConflict = errors.New("repositories: conflict")
	// ErrEmailTaken is returned when registration collides with an existing
	// account email. Unlike ErrConflict it is not retryable: the email is
	// already registered, so the caller must ask the user to sign in instead.
	ErrEmailTaken = errors.New("repositories: email already registered")
	// ErrInvalid is returned when input fails persistence-side validation.
	ErrInvalid = errors.New("repositories: invalid")
	// ErrForbidden is returned when the operation is rejected by policy or
	// by RLS.
	ErrForbidden = errors.New("repositories: forbidden")
	// ErrImmutable is returned when a write targets an append-only record.
	ErrImmutable = errors.New("repositories: immutable")
)
