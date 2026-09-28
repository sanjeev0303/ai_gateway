// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"strconv"
	"strings"

	"ai-governance-saas/backend/internal/domain"
)

// Authorize evaluates the snapshot's model allowlist/blocklist rules against
// the inspected request hints under deny-overrides: a blocklist match denies,
// an allowlist that omits the requested model denies, and no matching rules
// permit. An unknown model (empty hints.Model) skips model authorization: a
// transient inspection failure must not block org-wide traffic, and the
// upstream rejects unknown models anyway.
func (p PolicySnapshot) Authorize(hints InspectorHints) (bool, string) {
	if hints.Model == "" {
		return true, ""
	}

	var allowlists []PolicyRule
	for _, rule := range p.Rules {
		if rule.Type == domain.RuleTypeBlocklist && rule.Enabled && modelListed(ruleModels(rule), hints.Model) {
			return false, "model " + strconv.Quote(hints.Model) + " is blocked by policy"
		}
		if rule.Type == domain.RuleTypeAllowlist && rule.Enabled {
			allowlists = append(allowlists, rule)
		}
	}
	for _, rule := range allowlists {
		if modelListed(ruleModels(rule), hints.Model) {
			return true, ""
		}
	}
	if len(allowlists) > 0 {
		return false, "model " + strconv.Quote(hints.Model) + " is not allowed by policy"
	}
	return true, ""
}

// ruleModels returns the model strings a rule allows or blocks, read from the
// rule parameters ("models") with the conditions map as a fallback.
func ruleModels(rule PolicyRule) []string {
	raw, ok := rule.Actions["models"].([]any)
	if !ok {
		raw, ok = rule.Conditions["models"].([]any)
	}
	if !ok {
		return nil
	}
	models := make([]string, 0, len(raw))
	for _, m := range raw {
		if s, ok := m.(string); ok {
			models = append(models, s)
		}
	}
	return models
}

// modelListed reports whether the requested model appears in a rule's list,
// tolerating provider-qualified entries such as "anthropic/claude-sonnet-4-5".
func modelListed(list []string, model string) bool {
	for _, entry := range list {
		if entry == model || strings.HasSuffix(entry, "/"+model) {
			return true
		}
	}
	return false
}

// rateLimitFor returns the snapshot rate limit config for a scope, if any.
func (p PolicySnapshot) rateLimitFor(scope string) (RateLimitConfig, bool) {
	for _, cfg := range p.RateLimits {
		if cfg.Scope == scope {
			return cfg, true
		}
	}
	return RateLimitConfig{}, false
}

// budgetFor returns the snapshot budget config for a scope, if any.
func (p PolicySnapshot) budgetFor(scope string) (BudgetConfig, bool) {
	for _, cfg := range p.Budgets {
		if cfg.Scope == scope {
			return cfg, true
		}
	}
	return BudgetConfig{}, false
}
