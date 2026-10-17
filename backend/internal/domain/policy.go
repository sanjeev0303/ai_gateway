package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Policy enforcement modes, mirroring the CHECK in policies.
const (
	EnforcementPermissive = "permissive"
	EnforcementActive     = "active"
	EnforcementDisabled   = "disabled"
)

// Policy rule types, mirroring the CHECK in policy_rules.
const (
	RuleTypeRateLimit     = "rate_limit"
	RuleTypeTokenBudget   = "token_budget"
	RuleTypeBlocklist     = "blocklist"
	RuleTypeAllowlist     = "allowlist"
	RuleTypeContentFilter = "content_filter"
	RuleTypeBudget        = "budget"
)

// NewPolicy carries the caller-supplied fields for creating a policy.
type NewPolicy struct {
	Name        string
	Description string
	Enforcement string
	Priority    int32
}

type UpdatePolicy struct {
	Name        *string
	Description *string
	Enforcement *string
	Priority    *int32
}

// Policy groups rules that gate gateway traffic for an organization.
type Policy struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	Enforcement    string
	Priority       int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewPolicyRule carries the caller-supplied fields for creating a rule.
type NewPolicyRule struct {
	PolicyID    uuid.UUID
	RuleType    string
	Description string
	Conditions  []byte
	Parameters  []byte
	Enabled     bool
}

type UpdatePolicyRule struct {
	Description *string
	Conditions  []byte
	Parameters  []byte
	Enabled     *bool
}

// PolicyRule is a single enforceable condition within a policy.
type PolicyRule struct {
	ID             uuid.UUID
	PolicyID       uuid.UUID
	OrganizationID uuid.UUID
	RuleType       string
	Description    string
	Conditions     []byte
	Parameters     []byte
	Enabled        bool
	CreatedAt      time.Time
}

// FallbackPolicy carries the caller-supplied fields for provider fallback:
// when a requested model is missing on the preferred provider, the gateway
// may route to FallbackProvider with FallbackModel instead. Fallback is
// opt-in: a zero value disables it.
type FallbackPolicy struct {
	FallbackProvider string
	FallbackModel    string
}

// Validate reports whether the fallback policy is well-formed: the fallback
// provider must be a known provider kind, and the fallback model must be
// non-empty. A zero FallbackPolicy (both fields empty) is valid and means
// fallback is disabled.
func (p FallbackPolicy) Validate() error {
	if p.FallbackProvider == "" {
		return nil
	}
	if !IsProviderKind(p.FallbackProvider) {
		return errors.New("policy: fallback provider not a known provider kind")
	}
	if p.FallbackModel == "" {
		return errors.New("policy: fallback model required when fallback provider is set")
	}
	return nil
}

// BudgetScope represents the scope at which a budget is applied.
type BudgetScope string

const (
	BudgetScopeOrganization      BudgetScope = "organization"
	BudgetScopeTeam              BudgetScope = "team"
	BudgetScopeProject           BudgetScope = "project"
	BudgetScopeGatewayCredential BudgetScope = "gateway_credential"
	BudgetScopeServiceAccount    BudgetScope = "service_account"
)

// BudgetLimitType represents the type of limit a budget enforces.
type BudgetLimitType string

const (
	BudgetLimitTypeSpend    BudgetLimitType = "spend"
	BudgetLimitTypeTokens   BudgetLimitType = "tokens"
	BudgetLimitTypeRequests BudgetLimitType = "requests"
)

// BudgetWindow represents the time window for a budget.
type BudgetWindow string

const (
	BudgetWindowHourly  BudgetWindow = "hourly"
	BudgetWindowDaily   BudgetWindow = "daily"
	BudgetWindowWeekly  BudgetWindow = "weekly"
	BudgetWindowMonthly BudgetWindow = "monthly"
)

// NewBudget carries the caller-supplied fields for creating a budget.
type NewBudget struct {
	OrganizationID uuid.UUID
	Name           string
	Description    string
	Scope          BudgetScope
	ScopeID        uuid.UUID // team_id, project_id, credential_id, or service_account_id
	LimitType      BudgetLimitType
	LimitAmount    float64
	Window         BudgetWindow
	AlertThreshold float64 // percentage (0.0-1.0) at which to alert
	Enabled        bool
}

// UpdateBudget carries the caller-supplied fields for updating a budget.
type UpdateBudget struct {
	Name           *string
	Description    *string
	LimitAmount    *float64
	AlertThreshold *float64
	Enabled        *bool
}

// Budget represents a spend/token/request cap enforced on the gateway.
type Budget struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	Scope          BudgetScope
	ScopeID        uuid.UUID
	LimitType      BudgetLimitType
	LimitAmount    float64
	Window         BudgetWindow
	AlertThreshold float64
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// Computed fields (not stored)
	CurrentSpend    float64
	CurrentTokens   int64
	CurrentRequests int64
	WindowStart     time.Time
	WindowEnd       time.Time
}

// ModelAccessEntry represents a single model allow/block rule.
type ModelAccessEntry struct {
	Provider  string
	Model     string
	Allowed   bool   // true = allow, false = block
	Source    string // "allowlist" or "blocklist"
	PolicyID  uuid.UUID
	RuleID    uuid.UUID
	Priority  int32
	Effective bool // whether this rule takes effect (highest priority wins)
}

// ModelAccessPreviewRequest represents a request to preview model access changes.
type ModelAccessPreviewRequest struct {
	Provider string
	Model    string
	Action   string // "allow" or "block"
}

// ModelAccessPreviewResponse represents the result of a model access preview.
type ModelAccessPreviewResponse struct {
	Provider        string
	Model           string
	CurrentDecision string // "allow" or "block"
	PreviewDecision string // "allow" or "block"
	WouldChange     bool
	BlockingRules   []ModelAccessEntry
	AllowingRules   []ModelAccessEntry
}
