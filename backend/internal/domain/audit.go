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

// AuditLogFilter narrows a list of audit records. A nil pointer or the zero
// time means that dimension is unconstrained. ActorID matches either actor
// column; ActorType disambiguates which one it was.
type AuditLogFilter struct {
	ActorID      *uuid.UUID
	ActorType    *string
	Action       *string
	ResourceType *string
	ResourceID   *uuid.UUID
	From         time.Time
	To           time.Time
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
