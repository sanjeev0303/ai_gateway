package domain

import (
	"time"

	"github.com/google/uuid"
)

// ServiceSample is one poll of one service by the platform monitor.
type ServiceSample struct {
	Service    string
	SampledAt  time.Time
	Up         bool
	Error      string
	CPUPercent *float64
	RSSBytes   *int64
	HeapBytes  *int64
	Goroutines *int
	Detail     []byte
}

// ServiceLog is a warning or error line a service wrote.
type ServiceLog struct {
	ID       int64
	Service  string
	Level    string
	Message  string
	Attrs    []byte
	LoggedAt time.Time
}

type ServiceLogFilter struct {
	Service string
	// Level is "WARN", "ERROR" or empty for both.
	Level string
	// Search matches the message, case-insensitively.
	Search string
}

// Platform alert statuses and severities.
const (
	PlatformAlertOpen         = "open"
	PlatformAlertAcknowledged = "acknowledged"
	PlatformAlertResolved     = "resolved"

	PlatformAlertWarning  = "warning"
	PlatformAlertCritical = "critical"
)

// PlatformAlert is a problem the monitor found with a service or the worker.
// It stays one row while the problem lasts and resolves itself when it clears.
type PlatformAlert struct {
	ID             uuid.UUID
	Kind           string
	Service        string
	Severity       string
	Title          string
	Detail         string
	Status         string
	OpenedAt       time.Time
	LastSeenAt     time.Time
	AcknowledgedAt *time.Time
	AcknowledgedBy string
	ResolvedAt     *time.Time
	NotifiedAt     *time.Time
}

// NewPlatformAlert is a condition the monitor raises.
type NewPlatformAlert struct {
	Kind     string
	Service  string
	Severity string
	Title    string
	Detail   string
}

// TenantAuditEntry is one row of some organization's own audit log.
type TenantAuditEntry struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	ActorEmail       string
	Action           string
	ResourceType     string
	ResourceID       *uuid.UUID
	After            []byte
	IPAddress        *string
	CreatedAt        time.Time
}

// PlatformSecurityEvent is a security event with its organization's name.
type PlatformSecurityEvent struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Category         string
	Severity         string
	Title            string
	Detail           []byte
	Source           string
	CreatedAt        time.Time
}

// RevenueSummary is money actually taken: the sum of paid invoices.
type RevenueSummary struct {
	LifetimeMicros  int64
	Last30DMicros   int64
	ThisMonthMicros int64
	Payments        int64
	PayingOrgs      int64
}

type RevenueMonth struct {
	Month         time.Time
	RevenueMicros int64
	Payments      int64
}

// PlatformPayment is one paid invoice.
type PlatformPayment struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Number           string
	TotalMicros      int64
	Currency         string
	Provider         string
	PaymentID        string
	PaidAt           time.Time
}

// PayingCustomer is an organization that has paid at least once.
type PayingCustomer struct {
	OrganizationID   uuid.UUID
	OrganizationName string
	PlanName         string
	LifetimeMicros   int64
	Payments         int64
	LastPaidAt       time.Time
}
