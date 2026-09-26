package identity

import "errors"

// Sentinel errors returned by the identity service for failures that have no
// repository equivalent. Repository, validation, and authorization failures
// surface the port-level sentinels (ErrNotFound, ErrConflict, ErrInvalid,
// ErrForbidden) unchanged.
var (
	// ErrUnauthorized is returned when a presented credential (JWT or API
	// key) cannot be mapped to an active principal.
	ErrUnauthorized = errors.New("identity: unauthorized")
	// ErrInvalidInvitation is returned when an invitation token resolves to
	// an invitation that cannot be accepted (already revoked or consumed).
	ErrInvalidInvitation = errors.New("identity: invalid invitation")
	// ErrInvitationExpired is returned when an invitation has passed its
	// expiry.
	ErrInvitationExpired = errors.New("identity: invitation expired")
	// ErrInvitationEmailMismatch is returned when the authenticated user's
	// email differs from the address an invitation was issued to.
	ErrInvitationEmailMismatch = errors.New("identity: invitation email mismatch")
)
