package domain

import (
	"time"

	"github.com/google/uuid"
)

// Invitation statuses, mirroring the CHECK in invitations.
const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRevoked  = "revoked"
	InvitationStatusExpired  = "expired"
)

// NewInvitation carries the caller-supplied fields for creating an
// organization invitation. TokenHash is the sha256 hex digest of the raw
// invite token; the raw token is generated (and returned exactly once) by the
// application layer.
type NewInvitation struct {
	Email     string
	Role      string
	TokenHash string
	Expires   time.Time
	InvitedBy *uuid.UUID
}

// Invitation is a token-gated offer of an organization membership.
type Invitation struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	Role           string
	TokenHash      string
	Status         string
	ExpiresAt      time.Time
	InvitedBy      *uuid.UUID
	AcceptedBy     *uuid.UUID
	AcceptedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsExpired reports whether the invitation can no longer be accepted based on
// its expiry timestamp. Expired is a derived state; the row is still marked
// pending until a resolution action observes it.
func (i Invitation) IsExpired(now time.Time) bool {
	return i.ExpiresAt.Before(now)
}
