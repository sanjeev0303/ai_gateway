package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog is an immutable, append-only record of a control-plane action.
type AuditLog struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	ActorUserID           *uuid.UUID
	ActorServiceAccountID *uuid.UUID
	Action                string
	ResourceType          string
	ResourceID            *uuid.UUID
	Before                []byte
	After                 []byte
	IPAddress             *string
	RequestID             *string
	CreatedAt             time.Time
}

// NewAuditLog carries the caller-supplied fields for appending an audit
// record. The tenant (OrganizationID) is supplied to the repository call.
type NewAuditLog struct {
	ActorUserID           *uuid.UUID
	ActorServiceAccountID *uuid.UUID
	Action                string
	ResourceType          string
	ResourceID            *uuid.UUID
	Before                []byte
	After                 []byte
	IPAddress             *string
	RequestID             *string
}
