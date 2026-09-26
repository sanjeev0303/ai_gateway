// Package gatewayv2 contract tests for the policy authorization stage:
// allowlist/blocklist evaluation and the unknown-model skip.
package gatewayv2

import (
	"testing"

	"ai-governance-saas/backend/internal/domain"
)

// modelRule builds an enabled allowlist/blocklist rule for the given models.
func modelRule(typ string, models ...string) PolicyRule {
	list := make([]any, len(models))
	for i, m := range models {
		list[i] = m
	}
	return PolicyRule{Type: typ, Enabled: true, Actions: map[string]any{"models": list}}
}

func TestAuthorize_AllowlistedModelAdmitted(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5"),
	}}

	allowed, reason := p.Authorize(InspectorHints{Model: "claude-sonnet-4-5"})
	requireTrue(t, allowed, "allowlisted model must pass")
	requireTrue(t, reason == "", "no reason expected on allow, got %q", reason)
}

func TestAuthorize_UnlistedModelDenied(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5"),
	}}

	allowed, reason := p.Authorize(InspectorHints{Model: "gpt-4o"})
	requireTrue(t, !allowed, "unlisted model must be denied")
	requireTrue(t, reason != "", "denial must carry a reason")
}

func TestAuthorize_BlocklistedModelDenied(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeBlocklist, "gpt-4o"),
	}}

	allowed, reason := p.Authorize(InspectorHints{Model: "gpt-4o"})
	requireTrue(t, !allowed, "blocklisted model must be denied")
	requireTrue(t, reason != "", "denial must carry a reason")
}

func TestAuthorize_DenyOverridesAllowlist(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5", "gpt-4o"),
		modelRule(domain.RuleTypeBlocklist, "gpt-4o"),
	}}

	allowed, _ := p.Authorize(InspectorHints{Model: "gpt-4o"})
	requireTrue(t, !allowed, "blocklist must win over allowlist")
}

func TestAuthorize_QualifiedEntryMatchesBareModel(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeAllowlist, "anthropic/claude-sonnet-4-5"),
	}}

	allowed, _ := p.Authorize(InspectorHints{Model: "claude-sonnet-4-5"})
	requireTrue(t, allowed, "provider-qualified entry must match the bare model")
}

func TestAuthorize_UnknownModelSkips(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5"),
	}}

	allowed, reason := p.Authorize(InspectorHints{Model: ""})
	requireTrue(t, allowed, "unknown model must not be denied")
	requireTrue(t, reason == "", "no reason expected on skip, got %q", reason)
}

func TestAuthorize_NoModelRulesPermit(t *testing.T) {
	p := PolicySnapshot{Rules: []PolicyRule{
		modelRule(domain.RuleTypeRateLimit),
	}}

	allowed, _ := p.Authorize(InspectorHints{Model: "gpt-4o"})
	requireTrue(t, allowed, "orgs without model rules must keep working")
}

func TestAuthorize_DisabledRuleSkipped(t *testing.T) {
	rule := modelRule(domain.RuleTypeBlocklist, "gpt-4o")
	rule.Enabled = false
	p := PolicySnapshot{Rules: []PolicyRule{rule}}

	allowed, _ := p.Authorize(InspectorHints{Model: "gpt-4o"})
	requireTrue(t, allowed, "disabled blocklist must not deny")
}

func TestRateLimitFor_MatchingScopeReturned(t *testing.T) {
	p := PolicySnapshot{RateLimits: map[string]RateLimitConfig{
		"cred:infer": {Limit: 5, Window: 30 * 1e9, Scope: "credential"},
	}}

	cfg, ok := p.rateLimitFor("credential")
	requireTrue(t, ok, "credential-scoped config must be found")
	requireTrue(t, cfg.Limit == 5, "want limit 5, got %d", cfg.Limit)
}

func TestRateLimitFor_UnknownScopeMisses(t *testing.T) {
	p := PolicySnapshot{RateLimits: map[string]RateLimitConfig{
		"cred:infer": {Limit: 5, Window: 30 * 1e9, Scope: "credential"},
	}}

	_, ok := p.rateLimitFor("organization")
	requireTrue(t, !ok, "mismatched scope must not match")
}

func TestBudgetFor_MatchingScopeReturned(t *testing.T) {
	p := PolicySnapshot{Budgets: map[string]BudgetConfig{
		"org:budget": {Limit: 42, Window: 45 * 60 * 1e9, Scope: "organization"},
	}}

	cfg, ok := p.budgetFor("organization")
	requireTrue(t, ok, "organization-scoped config must be found")
	requireTrue(t, cfg.Limit == 42, "want limit 42, got %v", cfg.Limit)
}