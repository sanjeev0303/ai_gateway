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

// Invoice statuses, mirroring the CHECK in invoices. There is deliberately no
// 'paid' status: no payment processor is integrated, so no state here may imply
// that money moved.
const (
	InvoiceStatusOpen = "open"
	InvoiceStatusVoid = "void"
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

// OrganizationPlan is the organization's current plan and billing cycle. It
// holds only stored columns — deliberately no price, seats or billing contact.
type OrganizationPlan struct {
	PlanCode           string
	PlanName           string
	Status             string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
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

// Invoice is a local billing statement for one subscription period. It holds
// only stored columns — there is no processor reference, charge id, or PDF, and
// no field implies a payment was processed.
type Invoice struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID *uuid.UUID
	Number         string
	Status         string
	Currency       string
	TotalMicros    int64
	PeriodStart    time.Time
	PeriodEnd      time.Time
	IssuedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Lines          []InvoiceLine
}

// InvoiceLine is one itemisation row behind an invoice total.
type InvoiceLine struct {
	ID           uuid.UUID
	InvoiceID    uuid.UUID
	Description  string
	Quantity     int64
	UnitMicros   int64
	AmountMicros int64
	CreatedAt    time.Time
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

// AlertUpdate carries the mutable fields of an alert rule. AlertType is
// deliberately absent: the type selects which evaluator would read the rule, so
// it is fixed at creation rather than re-pointed at a different evaluator.
// Configuration must be a JSON object; the service validates it.
type AlertUpdate struct {
	Name          string
	Severity      string
	Configuration []byte
	Status        string
}

func ValidAlertSeverity(sev string) bool {
	return sev == AlertSeverityInfo || sev == AlertSeverityWarning || sev == AlertSeverityCritical
}

func ValidAlertStatus(st string) bool {
	return st == AlertStatusActive || st == AlertStatusMuted
}
