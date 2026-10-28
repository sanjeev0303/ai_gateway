package domain

import (
	"errors"
	"math"
	"strings"
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
	InvoiceStatusDraft         = "draft"
	InvoiceStatusOpen          = "open"
	InvoiceStatusVoid          = "void"
	InvoiceStatusPaid          = "paid"
	InvoiceStatusUncollectible = "uncollectible"
)

// Checkout session statuses.
const (
	CheckoutStatusPending   = "pending"
	CheckoutStatusCompleted = "completed"
	CheckoutStatusExpired   = "expired"
	CheckoutStatusCancelled = "cancelled"
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
	ID              uuid.UUID
	Code            string
	Name            string
	Description     string
	Status          string
	PriceMicros     int64
	Currency        string
	BillingInterval string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// GetPlanPrice returns the authoritative price and currency for standard plans.
func GetPlanPrice(planCode string) (int64, string) {
	switch planCode {
	case "enterprise":
		return 499_000_000, "usd" // $499.00
	case "pro":
		return 49_000_000, "usd" // $49.00
	default:
		return 0, "usd" // Free
	}
}

// Checkout tracks trusted server-side checkout state.
type Checkout struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	CheckoutSessionID string
	Provider          string
	PlanCode          string
	PlanID            *uuid.UUID
	Status            string // "pending", "completed", "expired", "cancelled"
	AmountMicros      int64
	Currency          string
	CustomerEmail     string
	CustomerID        string
	OrderID           string
	PaymentID         string
	Metadata          []byte
	ExpiresAt         time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewCheckout is the input for initiating a tracked checkout session.
type NewCheckout struct {
	CheckoutSessionID string
	Provider          string
	PlanCode          string
	PlanID            *uuid.UUID
	AmountMicros      int64
	Currency          string
	CustomerEmail     string
	CustomerID        string
	OrderID           string
	Metadata          []byte
	ExpiresAt         time.Time
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
	ID                             uuid.UUID  `json:"id"`
	PricingModelID                 uuid.UUID  `json:"pricing_model_id"`
	OrganizationID                 uuid.UUID  `json:"organization_id"`
	Version                        int32      `json:"version"`
	InputUnitPriceMicros           int64      `json:"input_unit_price_micros"`
	OutputUnitPriceMicros          int64      `json:"output_unit_price_micros"`
	CacheReadInputTokenPriceMicros int64      `json:"cache_read_input_token_price_micros"`
	EffectiveFrom                  time.Time  `json:"effective_from"`
	EffectiveTo                    *time.Time `json:"effective_to,omitempty"`
	CreatedAt                      time.Time  `json:"created_at"`
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
	Code            string
	Name            string
	Description     string
	PriceMicros     int64
	Currency        string
	BillingInterval string
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
	PricingModelID                 uuid.UUID
	Version                        int32
	InputUnitPriceMicros           int64
	OutputUnitPriceMicros          int64
	CacheReadInputTokenPriceMicros int64
	EffectiveFrom                  time.Time
	EffectiveTo                    *time.Time
}

// Pricing errors.
var (
	ErrNegativePrice              = errors.New("pricing: prices must be non-negative")
	ErrInvalidCurrency            = errors.New("pricing: currency must be a valid 3-letter ISO code")
	ErrUnsupportedTokenCategory   = errors.New("pricing: provider or model does not support prompt caching")
	ErrInvalidEffectiveWindow     = errors.New("pricing: effective_to must be strictly after effective_from")
	ErrPricingNotFound            = errors.New("pricing: pricing model or version not found")
	ErrPricingUnavailable         = errors.New("pricing: model pricing is unavailable")
	ErrInvalidTokenCount          = errors.New("pricing: token counts cannot be negative")
	ErrInvalidProviderKind        = errors.New("pricing: provider kind must be openai, anthropic, gemini, huggingface, azure_openai, or custom")
)

// ValidProviderKind reports whether kind is one of the supported provider kinds.
func ValidProviderKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "openai", "anthropic", "gemini", "huggingface", "azure_openai", "custom":
		return true
	default:
		return false
	}
}

// ModelSupportsCachedTokens reports whether a provider and model slug support prompt/context caching.
func ModelSupportsCachedTokens(providerKind, modelSlug string) bool {
	provider := strings.ToLower(strings.TrimSpace(providerKind))
	model := strings.ToLower(strings.TrimSpace(modelSlug))
	switch provider {
	case "anthropic":
		return strings.HasPrefix(model, "claude-3")
	case "openai":
		return strings.HasPrefix(model, "gpt-4o") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "gpt-4.5")
	case "gemini":
		return strings.Contains(model, "1.5") || strings.Contains(model, "2.0") || strings.Contains(model, "2.5") || strings.Contains(model, "flash") || strings.Contains(model, "pro")
	case "custom":
		return strings.Contains(model, "cache")
	default:
		return false
	}
}

// ProviderPricingSpec defines the pricing parameters for an individual provider model.
type ProviderPricingSpec struct {
	ProviderKind              string  `json:"provider_kind"`
	ModelSlug                 string  `json:"model_slug"`
	DisplayName               string  `json:"display_name"`
	InputRatePerM             float64 `json:"input_rate_per_m"`
	OutputRatePerM            float64 `json:"output_rate_per_m"`
	CachedReadRatePerM        float64 `json:"cached_read_rate_per_m"`
	SupportsCaching           bool    `json:"supports_caching"`
	DeepContextThreshold      int64   `json:"deep_context_threshold"`
	DeepContextInputRatePerM  float64 `json:"deep_context_input_rate_per_m"`
	DeepContextOutputRatePerM float64 `json:"deep_context_output_rate_per_m"`
}

// LookupCanonicalPricing returns the authoritative provider pricing specification for a model.
func LookupCanonicalPricing(providerKind, modelSlug string) (ProviderPricingSpec, bool) {
	provider := strings.ToLower(strings.TrimSpace(providerKind))
	slug := strings.ToLower(strings.TrimSpace(modelSlug))

	// 1. Anthropic models
	if provider == "anthropic" || strings.Contains(slug, "claude") {
		if strings.Contains(slug, "haiku") {
			return ProviderPricingSpec{
				ProviderKind: "anthropic", ModelSlug: modelSlug, DisplayName: "Claude 3.5 Haiku",
				InputRatePerM: 0.80, OutputRatePerM: 4.00, CachedReadRatePerM: 0.08, SupportsCaching: true,
			}, true
		}
		if strings.Contains(slug, "opus") {
			return ProviderPricingSpec{
				ProviderKind: "anthropic", ModelSlug: modelSlug, DisplayName: "Claude 3 Opus",
				InputRatePerM: 15.00, OutputRatePerM: 75.00, CachedReadRatePerM: 1.50, SupportsCaching: true,
			}, true
		}
		// Default Sonnet
		return ProviderPricingSpec{
			ProviderKind: "anthropic", ModelSlug: modelSlug, DisplayName: "Claude 3.5 Sonnet",
			InputRatePerM: 3.00, OutputRatePerM: 15.00, CachedReadRatePerM: 0.30, SupportsCaching: true,
		}, true
	}

	// 2. OpenAI models
	if provider == "openai" || strings.HasPrefix(slug, "gpt-") || strings.HasPrefix(slug, "o1") || strings.HasPrefix(slug, "o3") {
		if strings.Contains(slug, "mini") && strings.HasPrefix(slug, "gpt") {
			return ProviderPricingSpec{
				ProviderKind: "openai", ModelSlug: modelSlug, DisplayName: "GPT-4o Mini",
				InputRatePerM: 0.15, OutputRatePerM: 0.60, CachedReadRatePerM: 0.075, SupportsCaching: true,
			}, true
		}
		if strings.Contains(slug, "o1-mini") {
			return ProviderPricingSpec{
				ProviderKind: "openai", ModelSlug: modelSlug, DisplayName: "o1-mini",
				InputRatePerM: 3.00, OutputRatePerM: 12.00, CachedReadRatePerM: 1.50, SupportsCaching: true,
			}, true
		}
		if strings.HasPrefix(slug, "o1") {
			return ProviderPricingSpec{
				ProviderKind: "openai", ModelSlug: modelSlug, DisplayName: "o1",
				InputRatePerM: 15.00, OutputRatePerM: 60.00, CachedReadRatePerM: 7.50, SupportsCaching: true,
			}, true
		}
		if strings.Contains(slug, "o3-mini") {
			return ProviderPricingSpec{
				ProviderKind: "openai", ModelSlug: modelSlug, DisplayName: "o3-mini",
				InputRatePerM: 1.10, OutputRatePerM: 4.40, CachedReadRatePerM: 0.55, SupportsCaching: true,
			}, true
		}
		// Default GPT-4o
		return ProviderPricingSpec{
			ProviderKind: "openai", ModelSlug: modelSlug, DisplayName: "GPT-4o",
			InputRatePerM: 2.50, OutputRatePerM: 10.00, CachedReadRatePerM: 1.25, SupportsCaching: true,
		}, true
	}

	// 3. Google Gemini models
	if provider == "gemini" || strings.Contains(slug, "gemini") || strings.Contains(slug, "gemma") {
		if strings.Contains(slug, "pro") {
			return ProviderPricingSpec{
				ProviderKind: "gemini", ModelSlug: modelSlug, DisplayName: "Gemini 2.5 Pro",
				InputRatePerM: 1.25, OutputRatePerM: 5.00, CachedReadRatePerM: 0.3125, SupportsCaching: true,
				DeepContextThreshold: 128000, DeepContextInputRatePerM: 2.50, DeepContextOutputRatePerM: 10.00,
			}, true
		}
		if strings.Contains(slug, "flash-lite") {
			return ProviderPricingSpec{
				ProviderKind: "gemini", ModelSlug: modelSlug, DisplayName: "Gemini Flash-Lite",
				InputRatePerM: 0.0375, OutputRatePerM: 0.15, CachedReadRatePerM: 0.01, SupportsCaching: true,
				DeepContextThreshold: 128000, DeepContextInputRatePerM: 0.075, DeepContextOutputRatePerM: 0.30,
			}, true
		}
		// Default Flash
		return ProviderPricingSpec{
			ProviderKind: "gemini", ModelSlug: modelSlug, DisplayName: "Gemini 2.5 Flash",
			InputRatePerM: 0.075, OutputRatePerM: 0.30, CachedReadRatePerM: 0.01875, SupportsCaching: true,
			DeepContextThreshold: 128000, DeepContextInputRatePerM: 0.15, DeepContextOutputRatePerM: 0.60,
		}, true
	}

	// 4. Hugging Face models
	if provider == "huggingface" || strings.Contains(slug, "/") {
		if strings.Contains(slug, "deepseek") {
			return ProviderPricingSpec{
				ProviderKind: "huggingface", ModelSlug: modelSlug, DisplayName: "DeepSeek R1 Distill",
				InputRatePerM: 0.15, OutputRatePerM: 0.60, CachedReadRatePerM: 0.15, SupportsCaching: false,
			}, true
		}
		if strings.Contains(slug, "mistral") {
			return ProviderPricingSpec{
				ProviderKind: "huggingface", ModelSlug: modelSlug, DisplayName: "Mistral 7B Instruct",
				InputRatePerM: 0.05, OutputRatePerM: 0.05, CachedReadRatePerM: 0.05, SupportsCaching: false,
			}, true
		}
		if strings.Contains(slug, "qwen") {
			return ProviderPricingSpec{
				ProviderKind: "huggingface", ModelSlug: modelSlug, DisplayName: "Qwen 2.5 72B",
				InputRatePerM: 0.40, OutputRatePerM: 0.40, CachedReadRatePerM: 0.40, SupportsCaching: false,
			}, true
		}
		// Default Llama 3.3 70B
		return ProviderPricingSpec{
			ProviderKind: "huggingface", ModelSlug: modelSlug, DisplayName: "Llama 3.3 70B Instruct",
			InputRatePerM: 0.40, OutputRatePerM: 0.40, CachedReadRatePerM: 0.40, SupportsCaching: false,
		}, true
	}

	return ProviderPricingSpec{}, false
}

// CalculateProviderTokenCost calculates the exact cost in micro-dollars based on provider-specific pricing rules.
func CalculateProviderTokenCost(spec ProviderPricingSpec, promptTokens, completionTokens, cachedInputTokens int64) int64 {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	if cachedInputTokens < 0 {
		cachedInputTokens = 0
	}

	inputRate := spec.InputRatePerM
	outputRate := spec.OutputRatePerM

	// Handle deep context threshold (e.g. Gemini 128k boundary)
	if spec.DeepContextThreshold > 0 && promptTokens > spec.DeepContextThreshold {
		if spec.DeepContextInputRatePerM > 0 {
			inputRate = spec.DeepContextInputRatePerM
		}
		if spec.DeepContextOutputRatePerM > 0 {
			outputRate = spec.DeepContextOutputRatePerM
		}
	}

	nonCachedTokens := promptTokens
	cachedCost := 0.0
	if spec.SupportsCaching && cachedInputTokens > 0 {
		if cachedInputTokens > promptTokens {
			cachedInputTokens = promptTokens
		}
		nonCachedTokens = promptTokens - cachedInputTokens
		cachedCost = float64(cachedInputTokens) * spec.CachedReadRatePerM
	}

	inputCost := float64(nonCachedTokens) * inputRate
	outputCost := float64(completionTokens) * outputRate
	totalMicros := int64(math.Round(inputCost + cachedCost + outputCost))
	if totalMicros <= 0 && (promptTokens > 0 || completionTokens > 0) {
		totalMicros = 1
	}
	return totalMicros
}

// ResolveAndCalculateModelCost resolves provider-specific rates and computes request cost in micro-dollars.
func ResolveAndCalculateModelCost(providerKind, modelSlug string, promptTokens, completionTokens, cachedInputTokens int64) int64 {
	if spec, ok := LookupCanonicalPricing(providerKind, modelSlug); ok {
		return CalculateProviderTokenCost(spec, promptTokens, completionTokens, cachedInputTokens)
	}
	// Fallback to standard industry average (10 micros input, 30 micros output)
	total := promptTokens*10 + completionTokens*30
	if total <= 0 && (promptTokens > 0 || completionTokens > 0) {
		return 1
	}
	return total
}

// CalculateCost computes total request cost in micros given pricing parameters and token counts.
func CalculateCost(pv PricingVersion, promptTokens, completionTokens, cachedInputTokens int64) (int64, error) {
	if promptTokens < 0 || completionTokens < 0 || cachedInputTokens < 0 {
		return 0, ErrInvalidTokenCount
	}
	var inputCost int64
	if pv.CacheReadInputTokenPriceMicros > 0 && cachedInputTokens > 0 {
		nonCached := promptTokens - cachedInputTokens
		if nonCached < 0 {
			nonCached = 0
		}
		inputCost = (nonCached * pv.InputUnitPriceMicros) + (cachedInputTokens * pv.CacheReadInputTokenPriceMicros)
	} else {
		inputCost = promptTokens * pv.InputUnitPriceMicros
	}
	outputCost := completionTokens * pv.OutputUnitPriceMicros
	return inputCost + outputCost, nil
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
