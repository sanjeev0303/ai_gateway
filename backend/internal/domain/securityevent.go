package domain

import (
	"time"

	"github.com/google/uuid"
)

// SecurityEventCategory names the kind of condition the platform detected
// about itself. It is deliberately narrower than an audit action: audit_logs
// already records who changed what, so these categories cover only what the
// audit ledger cannot observe.
type SecurityEventCategory string

const (
	// SecurityEventProviderIncident is a provider's observed reachability
	// changing, as seen by a live probe.
	SecurityEventProviderIncident SecurityEventCategory = "provider_incident"
	// SecurityEventPipelineFailure is a background rollup failing for one
	// organization.
	SecurityEventPipelineFailure SecurityEventCategory = "pipeline_failure"
	// SecurityEventPolicyViolation is a request being denied by a policy rule, budget, or rate limit.
	SecurityEventPolicyViolation SecurityEventCategory = "policy_violation"
)

// Valid reports whether c is one of the supported categories.
func (c SecurityEventCategory) Valid() bool {
	switch c {
	case SecurityEventProviderIncident, SecurityEventPipelineFailure, SecurityEventPolicyViolation:
		return true
	default:
		return false
	}
}

// SecurityEventSeverity is the tone a reader should render an event with.
type SecurityEventSeverity string

const (
	SecurityEventSeverityInfo     SecurityEventSeverity = "info"
	SecurityEventSeverityWarning  SecurityEventSeverity = "warning"
	SecurityEventSeverityCritical SecurityEventSeverity = "critical"
)

// Valid reports whether s is one of the supported severities.
func (s SecurityEventSeverity) Valid() bool {
	switch s {
	case SecurityEventSeverityInfo, SecurityEventSeverityWarning, SecurityEventSeverityCritical:
		return true
	default:
		return false
	}
}

// SecurityEvent is an append-only record of something the platform detected
// about itself. The table has no updated_at and rejects UPDATE and DELETE, so a
// row is a statement about one moment and cannot later be revised.
type SecurityEvent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Category       SecurityEventCategory
	Severity       SecurityEventSeverity
	Title          string
	Detail         []byte
	Source         string
	CreatedAt      time.Time
}

// SecurityEventFilter narrows a list of events. A nil pointer or zero time
// leaves that dimension unconstrained.
type SecurityEventFilter struct {
	Category *SecurityEventCategory
	Severity *SecurityEventSeverity
	From     time.Time
	To       time.Time
}

// NewSecurityEvent carries the caller-supplied fields for appending an event.
// The tenant (OrganizationID) is supplied to the repository call, never here,
// so a caller cannot record an event against another organization.
type NewSecurityEvent struct {
	Category SecurityEventCategory
	Severity SecurityEventSeverity
	Title    string
	Detail   []byte
	Source   string
}
