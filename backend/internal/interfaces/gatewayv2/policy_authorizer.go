// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
)

// Authorize evaluates the snapshot's governance rules (credential scopes,
// model allowlists/blocklists, content size constraints, and provider access)
// against the inspected request hints and credential under deterministic precedence
// and deny-overrides.
func (p PolicySnapshot) Authorize(hints InspectorHints, cred ...GatewayCredential) (bool, string) {
	var c GatewayCredential
	hasCred := len(cred) > 0
	if hasCred {
		c = cred[0]

		// 1. Credential status checks (fail-closed)
		if c.Status == "revoked" {
			return false, "credential is revoked"
		}
		if c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt) {
			return false, "credential has expired"
		}

		// 2. Credential scope restrictions:
		// A gateway credential must have either "gateway" or "all" in its granted scopes.
		if len(c.Scopes) > 0 {
			hasGatewayScope := false
			for _, s := range c.Scopes {
				if s == "gateway" || s == "all" {
					hasGatewayScope = true
					break
				}
			}
			if !hasGatewayScope {
				return false, "credential lacks gateway scope"
			}
		}

		// 2b. Provider restrictions on credential
		if len(c.AllowedProviders) > 0 && hints.Provider != "" {
			providerAllowed := false
			for _, p := range c.AllowedProviders {
				if strings.EqualFold(p, hints.Provider) {
					providerAllowed = true
					break
				}
			}
			if !providerAllowed {
				return false, fmt.Sprintf("credential is not authorized for provider %q", hints.Provider)
			}
		}

		// 2c. Model restrictions on credential
		if len(c.AllowedModels) > 0 && hints.Model != "" {
			modelAllowed := false
			for _, m := range c.AllowedModels {
				if m == hints.Model {
					modelAllowed = true
					break
				}
			}
			if !modelAllowed {
				return false, fmt.Sprintf("credential is not authorized for model %q", hints.Model)
			}
		}
	}

	// 3. Content size bounds enforcement (max input / output tokens)
	for _, rule := range p.Rules {
		if !rule.Enabled {
			continue
		}
		if hasCred && !ruleMatchesScope(rule, c) {
			continue
		}

		maxInput := ruleSizeParam(rule, "max_input_tokens")
		if maxInput > 0 && hints.EstimatedInputTokens > maxInput {
			return false, fmt.Sprintf("request input tokens (%d) exceed maximum allowed limit of %d", hints.EstimatedInputTokens, maxInput)
		}

		maxOutput := ruleSizeParam(rule, "max_output_tokens")
		if maxOutput > 0 && hints.EstimatedOutputTokens > maxOutput {
			return false, fmt.Sprintf("request output tokens (%d) exceed maximum allowed limit of %d", hints.EstimatedOutputTokens, maxOutput)
		}
	}

	// 4. Model and Provider Access Control
	if hints.Model == "" {
		return true, ""
	}

	// Collect rules matching credential scope
	var matchingRules []PolicyRule
	for _, rule := range p.Rules {
		if !rule.Enabled {
			continue
		}
		if rule.Type != domain.RuleTypeAllowlist && rule.Type != domain.RuleTypeBlocklist {
			continue
		}
		if hasCred && !ruleMatchesScope(rule, c) {
			continue
		}
		matchingRules = append(matchingRules, rule)
	}

	// Deterministic specificity evaluation:
	// Rank 5: Credential-specific
	// Rank 4: User or Service-Account specific
	// Rank 3: Project-specific
	// Rank 2: Team-specific
	// Rank 1: Organization-wide
	for rank := 5; rank >= 1; rank-- {
		var rankRules []PolicyRule
		for _, r := range matchingRules {
			if ruleSpecificityRank(r) == rank {
				rankRules = append(rankRules, r)
			}
		}
		if len(rankRules) == 0 {
			continue
		}

		// Check Blocklists first at this rank (Deny-overrides)
		for _, r := range rankRules {
			if r.Type == domain.RuleTypeBlocklist && modelListed(ruleModels(r), hints.Model) {
				return false, "model " + strconv.Quote(hints.Model) + " is blocked by policy"
			}
		}

		// Check Allowlists at this rank
		var allowlists []PolicyRule
		for _, r := range rankRules {
			if r.Type == domain.RuleTypeAllowlist {
				allowlists = append(allowlists, r)
			}
		}
		if len(allowlists) > 0 {
			for _, r := range allowlists {
				if modelListed(ruleModels(r), hints.Model) {
					return true, ""
				}
			}
			// If allowlists exist at this scope level and none matched, deny
			return false, "model " + strconv.Quote(hints.Model) + " is not allowed by policy"
		}
	}

	return true, ""
}

// ruleMatchesScope checks whether a rule's scope matches the request's gateway credential.
func ruleMatchesScope(rule PolicyRule, cred GatewayCredential) bool {
	scope := rule.Scope
	if scope == "" || scope == "organization" || scope == "org" {
		return true
	}
	switch scope {
	case "team":
		return cred.TeamID != nil && cred.TeamID.String() == rule.ScopeID
	case "project":
		return cred.ProjectID != nil && cred.ProjectID.String() == rule.ScopeID
	case "user":
		return cred.SubjectType == "user" && cred.SubjectID.String() == rule.ScopeID
	case "service_account":
		return cred.SubjectType == "service_account" && cred.SubjectID.String() == rule.ScopeID
	case "credential", "gateway_credential":
		return cred.ID.String() == rule.ScopeID
	default:
		return true
	}
}

// ruleSpecificityRank returns the precedence rank for deterministic resolution.
func ruleSpecificityRank(rule PolicyRule) int {
	switch rule.Scope {
	case "credential", "gateway_credential":
		return 5
	case "user", "service_account":
		return 4
	case "project":
		return 3
	case "team":
		return 2
	case "organization", "org", "":
		return 1
	default:
		return 1
	}
}

// ruleSizeParam extracts a numeric limit from actions or conditions.
func ruleSizeParam(rule PolicyRule, key string) int {
	if v, ok := rule.Actions[key]; ok {
		if n := int(toFloat64(v)); n > 0 {
			return n
		}
	}
	if v, ok := rule.Conditions[key]; ok {
		if n := int(toFloat64(v)); n > 0 {
			return n
		}
	}
	return 0
}

// ruleModels returns the model strings a rule allows or blocks, read from the
// rule parameters ("models", "model", "provider") with the conditions map as a fallback.
func ruleModels(rule PolicyRule) []string {
	var models []string

	extract := func(m map[string]any) {
		if raw, ok := m["models"].([]any); ok {
			for _, item := range raw {
				if s, ok := item.(string); ok && s != "" {
					models = append(models, s)
				}
			}
		}
		if s, ok := m["model"].(string); ok && s != "" {
			models = append(models, s)
		}
		if prov, ok := m["provider"].(string); ok && prov != "" && len(models) == 0 {
			models = append(models, prov+"/*")
		}
	}

	extract(rule.Actions)
	if len(models) == 0 {
		extract(rule.Conditions)
	}

	return models
}

// modelListed reports whether the requested model appears in a rule's list,
// tolerating provider-qualified entries such as "anthropic/claude-sonnet-4-5",
// exact matches, and wildcard patterns like "openai/*" or "*".
func modelListed(list []string, model string) bool {
	for _, entry := range list {
		if entry == "*" {
			return true
		}
		if entry == model {
			return true
		}
		if strings.HasSuffix(entry, "/"+model) {
			return true
		}
		if strings.HasSuffix(model, "/"+entry) {
			return true
		}
		// Wildcard provider check: e.g. "openai/*"
		if strings.HasSuffix(entry, "/*") {
			prefix := strings.TrimSuffix(entry, "/*")
			if strings.HasPrefix(model, prefix+"/") {
				return true
			}
		}
		// Prefix wildcard: e.g. "claude-*"
		if strings.HasSuffix(entry, "*") {
			prefix := strings.TrimSuffix(entry, "*")
			if strings.HasPrefix(model, prefix) || strings.Contains(model, "/"+prefix) {
				return true
			}
		}
	}
	return false
}

// applicableRateLimits returns all rate limit configs that apply to a credential.
func (p PolicySnapshot) applicableRateLimits(cred GatewayCredential) []RateLimitConfig {
	var out []RateLimitConfig
	seen := make(map[string]bool)

	// Direct credential-level rate limit override if configured
	if cred.RateLimit != nil && *cred.RateLimit > 0 {
		key := "credential_direct:" + cred.ID.String()
		out = append(out, RateLimitConfig{
			Key:     key,
			Limit:   *cred.RateLimit,
			Window:  time.Minute,
			Scope:   "credential",
			ScopeID: cred.ID.String(),
		})
		seen[key] = true
	}

	addIfMatch := func(cfg RateLimitConfig) {
		if cfg.Limit <= 0 || seen[cfg.Key] {
			return
		}
		switch cfg.Scope {
		case "credential", "gateway_credential":
			if cfg.ScopeID == "" || cfg.ScopeID == cred.ID.String() {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "user":
			if cred.SubjectType == "user" && (cfg.ScopeID == "" || cfg.ScopeID == cred.SubjectID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "service_account":
			if cred.SubjectType == "service_account" && (cfg.ScopeID == "" || cfg.ScopeID == cred.SubjectID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "project":
			if cred.ProjectID != nil && (cfg.ScopeID == "" || cfg.ScopeID == cred.ProjectID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "team":
			if cred.TeamID != nil && (cfg.ScopeID == "" || cfg.ScopeID == cred.TeamID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "organization", "org", "":
			out = append(out, cfg)
			seen[cfg.Key] = true
		}
	}

	for _, cfg := range p.RateLimits {
		addIfMatch(cfg)
	}

	return out
}

// applicableBudgets returns all budget configs that apply to a credential.
func (p PolicySnapshot) applicableBudgets(cred GatewayCredential) []BudgetConfig {
	var out []BudgetConfig
	seen := make(map[string]bool)

	// Direct credential-level budget limit override if configured
	if cred.BudgetLimit != nil && *cred.BudgetLimit > 0 {
		key := "credential_direct:" + cred.ID.String()
		out = append(out, BudgetConfig{
			Key:       key,
			Limit:     *cred.BudgetLimit,
			Window:    30 * 24 * time.Hour,
			Scope:     "credential",
			ScopeID:   cred.ID.String(),
			LimitType: "spend",
		})
		seen[key] = true
	}

	addIfMatch := func(cfg BudgetConfig) {
		if cfg.Limit <= 0 || seen[cfg.Key] {
			return
		}
		switch cfg.Scope {
		case "credential", "gateway_credential":
			if cfg.ScopeID == "" || cfg.ScopeID == cred.ID.String() {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "service_account":
			if cred.SubjectType == "service_account" && (cfg.ScopeID == "" || cfg.ScopeID == cred.SubjectID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "project":
			if cred.ProjectID != nil && (cfg.ScopeID == "" || cfg.ScopeID == cred.ProjectID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "team":
			if cred.TeamID != nil && (cfg.ScopeID == "" || cfg.ScopeID == cred.TeamID.String()) {
				out = append(out, cfg)
				seen[cfg.Key] = true
			}
		case "organization", "org", "":
			out = append(out, cfg)
			seen[cfg.Key] = true
		}
	}

	for _, cfg := range p.Budgets {
		addIfMatch(cfg)
	}

	return out
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
