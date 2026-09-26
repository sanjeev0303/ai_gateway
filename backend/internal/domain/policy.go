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
