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
	SubscriptionStatusUnpaid     = "unpaid"
)

// Billing event statuses, mirroring the CHECK in billing_events.
const (
	BillingEventStatusPending   = "pending"
	BillingEventStatusSucceeded = "succeeded"
	BillingEventStatusFailed    = "failed"
)

// Invoice statuses, mirroring the CHECK in invoices.
const (
	InvoiceStatusOpen = "open"
	InvoiceStatusVoid = "void"
	InvoiceStatusPaid = "paid"
)


// Alert types, severities and statuses, mirroring the CHECKs in alerts.
const (
	AlertTypeSpendLimit        = "spend_limit"
	AlertTypeSpendThreshold    = "spend_threshold"
	AlertTypeSpendExceeded     = "spend_exceeded"
	AlertTypeBudgetWarning     = "budget_warning"
	AlertTypeBudgetExhaustion  = "budget_exhaustion"
	AlertTypeRateLimit         = "rate_limit"
	AlertTypeRateLimitSpike    = "rate_limit_spike"
	AlertTypeErrorRate         = "error_rate"
	AlertTypeErrorRateSpike    = "error_rate_spike"
	AlertTypeAnomaly           = "anomaly"
	AlertTypeUsageAnomaly      = "usage_anomaly"
	AlertTypeUsageSpike        = "usage_spike"
	AlertTypeSpendSpike        = "spend_spike"
	AlertTypeProviderFailure   = "provider_failure"
	AlertTypeCredentialFailure = "credential_failure"
	AlertTypePolicyViolation   = "policy_violation"
	AlertTypePolicyDenialSpike = "policy_denial_spike"
)

const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityCritical = "critical"
)

const (
	AlertStatusActive   = "active"
	AlertStatusMuted    = "muted"
	AlertStatusResolved = "resolved"
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
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	PlanID                 uuid.UUID
	Status                 string
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	CancelAtPeriodEnd      bool
	PaymentProvider        string
	ProviderCustomerID     *string
	ProviderSubscriptionID *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// OrganizationPlan is the organization's current plan and billing cycle.
type OrganizationPlan struct {
	PlanCode               string
	PlanName               string
	Status                 string
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	CancelAtPeriodEnd      bool
	PaymentProvider        string
	ProviderCustomerID     *string
	ProviderSubscriptionID *string
}

// PlanEntitlements defines the operational quotas and access tiers for a subscription plan.
type PlanEntitlements struct {
	PlanCode            string   `json:"plan_code"`
	Name                string   `json:"name"`
	MonthlyRequestQuota int64    `json:"monthly_request_quota"`
	AllowedProviders    []string `json:"allowed_providers"`
	MaxProjects         int      `json:"max_projects"`
	MaxMembers          int      `json:"max_members"`
	RetentionDays       int      `json:"retention_days"`
	SupportsGuardrails  bool     `json:"supports_guardrails"`
	SupportsFullPrivacy bool     `json:"supports_full_privacy"`
}

// GetPlanEntitlements returns the governance limits and access flags for a plan code.
func GetPlanEntitlements(planCode string) PlanEntitlements {
	switch planCode {
	case "enterprise":
		return PlanEntitlements{
			PlanCode:            "enterprise",
			Name:                "Enterprise Tier",
			MonthlyRequestQuota: 100_000_000,
			AllowedProviders:    []string{"openai", "anthropic", "gemini", "azure_openai", "custom", "huggingface"},
			MaxProjects:         1000,
			MaxMembers:          500,
			RetentionDays:       365,
			SupportsGuardrails:  true,
			SupportsFullPrivacy: true,
		}
	case "pro":
		return PlanEntitlements{
			PlanCode:            "pro",
			Name:                "Pro Tier",
			MonthlyRequestQuota: 1_000_000,
			AllowedProviders:    []string{"openai", "anthropic", "gemini", "huggingface"},
			MaxProjects:         50,
			MaxMembers:          25,
			RetentionDays:       30,
			SupportsGuardrails:  true,
			SupportsFullPrivacy: true,
		}
	default:
		return PlanEntitlements{
			PlanCode:            "free",
			Name:                "Free Tier",
			MonthlyRequestQuota: 10_000,
			AllowedProviders:    []string{"openai", "gemini"},
			MaxProjects:         3,
			MaxMembers:          5,
			RetentionDays:       7,
			SupportsGuardrails:  false,
			SupportsFullPrivacy: false,
		}
	}
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

// Invoice is a billing statement for one subscription period or payment event.
type Invoice struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	SubscriptionID  *uuid.UUID
	Number          string
	Status          string
	Currency        string
	TotalMicros     int64
	PeriodStart     time.Time
	PeriodEnd       time.Time
	IssuedAt        time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PaymentProvider *string
	PaymentID       *string
	PaidAt          *time.Time
	Lines           []InvoiceLine
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
	return st == AlertStatusActive || st == AlertStatusMuted || st == AlertStatusResolved
}

// AlertEvent records an occurrence when an alert rule condition was triggered or resolved.
type AlertEvent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	AlertID        uuid.UUID
	AlertType      string
	Severity       string
	Status         string // triggered, resolved, acknowledged
	Message        string
	Context        []byte
	TriggeredAt    time.Time
	ResolvedAt     *time.Time
	CreatedAt      time.Time
}

// NewAlertEvent carries input data for recording an alert occurrence.
type NewAlertEvent struct {
	AlertID     uuid.UUID
	AlertType   string
	Severity    string
	Status      string
	Message     string
	Context     []byte
	TriggeredAt time.Time
	ResolvedAt  *time.Time
}

// InAppNotification represents an in-app alert or notification for an organization.
type InAppNotification struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	AlertID        *uuid.UUID
	AlertEventID   *uuid.UUID
	Title          string
	Message        string
	Severity       string
	Read           bool
	CreatedAt      time.Time
}

// NewInAppNotification carries fields to create a notification.
type NewInAppNotification struct {
	AlertID      *uuid.UUID
	AlertEventID *uuid.UUID
	Title        string
	Message      string
	Severity     string
}
