package domain

import (
	"time"

	"github.com/google/uuid"
)

// Platform admin views read across every organization. They back the
// super-admin console only; nothing tenant-facing uses them.

// PlatformOverview is the headline state of the whole platform.
type PlatformOverview struct {
	OrgsTotal         int64
	OrgsSuspended     int64
	UsersTotal        int64
	UsersDisabled     int64
	Requests          int64
	FailedRequests    int64
	Tokens            int64
	CostMicros        int64
	ActiveKeys        int64
	PaidSubscriptions int64
	MRRMicros         int64
}

// PlatformOrganization is one organization as the platform sees it.
type PlatformOrganization struct {
	ID                 uuid.UUID
	Name               string
	Slug               string
	Status             string
	CreatedAt          time.Time
	PlanCode           string
	PlanName           string
	SubscriptionStatus string
	Members            int64
	ActiveKeys         int64
	Requests           int64
	CostMicros         int64
	LastRequestAt      *time.Time
}

// PlatformOrgMember is a member of an organization, with the user's own status.
type PlatformOrgMember struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Role        string
	Status      string
	JoinedAt    time.Time
	LastLoginAt *time.Time
}

// PlatformUser is one user across all their organizations.
type PlatformUser struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Status      string
	CreatedAt   time.Time
	LastLoginAt *time.Time
	OrgCount    int64
	OrgNames    string
}

// PlatformUsageDay is one UTC day of platform traffic.
type PlatformUsageDay struct {
	Day            time.Time
	Requests       int64
	FailedRequests int64
	Tokens         int64
	CostMicros     int64
}

// PlatformUsageRow is traffic grouped by one dimension (provider, model or
// organization).
type PlatformUsageRow struct {
	Key          string
	Label        string
	Requests     int64
	Tokens       int64
	CostMicros   int64
	AvgLatencyMs int64
}

// PlatformPlanSummary counts live subscriptions on one plan.
type PlatformPlanSummary struct {
	PlanCode      string
	PlanName      string
	PriceMicros   int64
	Currency      string
	Subscriptions int64
}

// PlatformInvoice is an invoice with its organization's name.
type PlatformInvoice struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Number           string
	Status           string
	Currency         string
	TotalMicros      int64
	IssuedAt         time.Time
}

// PlatformAuditEntry is one platform admin action.
type PlatformAuditEntry struct {
	ID               uuid.UUID
	ActorUserID      *uuid.UUID
	ActorEmail       string
	Action           string
	TargetType       string
	TargetID         *uuid.UUID
	OrganizationID   *uuid.UUID
	OrganizationName string
	Reason           string
	Detail           []byte
	CreatedAt        time.Time
}

// NewPlatformAuditEntry is what a platform admin action records.
type NewPlatformAuditEntry struct {
	ActorUserID    uuid.UUID
	ActorEmail     string
	Action         string
	TargetType     string
	TargetID       *uuid.UUID
	OrganizationID *uuid.UUID
	Reason         string
	Detail         []byte
}

// PlatformOutbox is the backlog of the durable event outbox.
type PlatformOutbox struct {
	Pending       int64
	Retrying      int64
	OldestPending *time.Time
	// OldestAgeSeconds is measured on the database clock, so a skewed web or
	// API host cannot make a healthy queue look stuck.
	OldestAgeSeconds int64
	LastError        string
}

// PlatformCredentialRef points at a gateway key in some organization.
type PlatformCredentialRef struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}
