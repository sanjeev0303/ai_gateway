package domain

import (
	"time"

	"github.com/google/uuid"
)

// Plan and pricing statuses, mirroring their CHECKs.
const (
	PlanStatusActive   = "active"
	PlanStatusInactive = "inactive"
)

const (
	PricingModelStatusActive   = "active"
	PricingModelStatusInactive = "inactive"
)

// Subscription statuses, mirroring the CHECK in subscriptions.
const (
	SubscriptionStatusActive     = "active"
	SubscriptionStatusTrialing   = "trialing"
	SubscriptionStatusCanceled   = "canceled"
	SubscriptionStatusPastDue    = "past_due"
	SubscriptionStatusIncomplete = "incomplete"
)

// Billing event statuses, mirroring the CHECK in billing_events.
const (
	BillingEventStatusPending   = "pending"
	BillingEventStatusSucceeded = "succeeded"
	BillingEventStatusFailed    = "failed"
)

// Alert types, severities and statuses, mirroring the CHECKs in alerts.
const (
	AlertTypeSpendLimit = "spend_limit"
	AlertTypeRateLimit  = "rate_limit"
	AlertTypeErrorRate  = "error_rate"
	AlertTypeAnomaly    = "anomaly"
)

const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityCritical = "critical"
)

const (
	AlertStatusActive = "active"
	AlertStatusMuted  = "muted"
)

// Plan is a sellable subscription tier.
type Plan struct {
	ID          uuid.UUID
	Code        string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Subscription links an organization to a plan with billing period dates.
type Subscription struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	PlanID             uuid.UUID
	Status             string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BillingEvent is an append-only ledger of billing transactions.
type BillingEvent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	BillingEventID string
	EventType      string
	AmountMicros   *int64
	Currency       string
	Status         string
	Payload        []byte
	CreatedAt      time.Time
}

// PricingModel declares per-provider pricing for one model.
type PricingModel struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ModelSlug      string
	ProviderKind   string
	DisplayName    string
	Currency       string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PricingVersion is one versioned price sheet within a pricing model.
type PricingVersion struct {
	ID                    uuid.UUID
	PricingModelID        uuid.UUID
	OrganizationID        uuid.UUID
	Version               int32
	InputUnitPriceMicros  int64
	OutputUnitPriceMicros int64
	EffectiveFrom         time.Time
	CreatedAt             time.Time
}

// Alert is a configured notification rule.
type Alert struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	AlertType       string
	Name            string
	Severity        string
	Configuration   []byte
	Status          string
	LastTriggeredAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewPlan carries the caller-supplied fields for creating a plan. Status is
// defaulted by the database to 'active'.
type NewPlan struct {
	Code        string
	Name        string
	Description string
}

// NewSubscription carries the caller-supplied fields for subscribing an
// organization to a plan. Status is defaulted by the database to 'active'.
type NewSubscription struct {
	PlanID             uuid.UUID
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
}

// NewBillingEvent carries the caller-supplied fields for recording a billing
// event. Status is defaulted by the database to 'pending'.
type NewBillingEvent struct {
	BillingEventID string
	EventType      string
	AmountMicros   *int64
	Currency       string
	Payload        []byte
}

// NewPricingModel carries the caller-supplied fields for upserting a pricing
// model. Status is defaulted by the database to 'active'.
type NewPricingModel struct {
	ModelSlug    string
	ProviderKind string
	DisplayName  string
	Currency     string
}

// NewPricingVersion carries the caller-supplied fields for creating a pricing
// version within a pricing model.
type NewPricingVersion struct {
	PricingModelID        uuid.UUID
	Version               int32
	InputUnitPriceMicros  int64
	OutputUnitPriceMicros int64
	EffectiveFrom         time.Time
}

// NewAlert carries the caller-supplied fields for creating an alert. Status
// is defaulted by the database to 'active'; severity is an operational
// setting and is caller-supplied.
type NewAlert struct {
	AlertType     string
	Name          string
	Severity      string
	Configuration []byte
}
