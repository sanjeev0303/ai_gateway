package controlapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"

	"github.com/google/uuid"
)

type policyDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Enforcement    string `json:"enforcement"`
	Priority       int32  `json:"priority"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type policyRuleDTO struct {
	ID             string         `json:"id"`
	PolicyID       string         `json:"policy_id"`
	OrganizationID string         `json:"organization_id"`
	RuleType       string         `json:"rule_type"`
	Description    string         `json:"description"`
	Conditions     map[string]any `json:"conditions"`
	Parameters     map[string]any `json:"parameters"`
	Enabled        bool           `json:"enabled"`
	CreatedAt      string         `json:"created_at"`
}

func toPolicyDTO(p domain.Policy) policyDTO {
	return policyDTO{
		ID:             p.ID.String(),
		OrganizationID: p.OrganizationID.String(),
		Name:           p.Name,
		Description:    p.Description,
		Enforcement:    p.Enforcement,
		Priority:       p.Priority,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}
}

func toPolicyRuleDTO(r domain.PolicyRule) policyRuleDTO {
	var c map[string]any
	var p map[string]any
	if len(r.Conditions) > 0 {
		_ = json.Unmarshal(r.Conditions, &c)
	} else {
		c = map[string]any{}
	}
	if len(r.Parameters) > 0 {
		_ = json.Unmarshal(r.Parameters, &p)
	} else {
		p = map[string]any{}
	}
	return policyRuleDTO{
		ID:             r.ID.String(),
		PolicyID:       r.PolicyID.String(),
		OrganizationID: r.OrganizationID.String(),
		RuleType:       r.RuleType,
		Description:    r.Description,
		Conditions:     c,
		Parameters:     p,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
	}
}

// Handlers

func (a *api) createPolicy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enforcement string `json:"enforcement"`
		Priority    int32  `json:"priority"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	actor, _ := actorFrom(r.Context())
	pol, err := a.policy.CreatePolicy(r.Context(), actor, domain.NewPolicy{
		Name:        in.Name,
		Description: in.Description,
		Enforcement: in.Enforcement,
		Priority:    in.Priority,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toPolicyDTO(pol))
}

func (a *api) listPolicies(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	policies, err := a.policy.ListPolicies(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	out := make([]policyDTO, 0, len(policies))
	for _, p := range policies {
		out = append(out, toPolicyDTO(p))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (a *api) getPolicy(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	pol, err := a.policy.GetPolicy(r.Context(), actor.OrganizationID, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPolicyDTO(pol))
}

func (a *api) updatePolicy(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	var in domain.UpdatePolicy
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	pol, err := a.policy.UpdatePolicy(r.Context(), actor, id, in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPolicyDTO(pol))
}

func (a *api) deletePolicy(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	if err := a.policy.DeletePolicy(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) createPolicyRule(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	policyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	var in struct {
		RuleType    string         `json:"rule_type"`
		Description string         `json:"description"`
		Conditions  map[string]any `json:"conditions"`
		Parameters  map[string]any `json:"parameters"`
		Enabled     bool           `json:"enabled"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	condBytes, _ := json.Marshal(in.Conditions)
	paramBytes, _ := json.Marshal(in.Parameters)
	
	rule, err := a.policy.CreateRule(r.Context(), actor, domain.NewPolicyRule{
		PolicyID:    policyID,
		RuleType:    in.RuleType,
		Description: in.Description,
		Conditions:  condBytes,
		Parameters:  paramBytes,
		Enabled:     in.Enabled,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toPolicyRuleDTO(rule))
}

func (a *api) listPolicyRules(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	policyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, policyID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	out := make([]policyRuleDTO, 0, len(rules))
	for _, r := range rules {
		out = append(out, toPolicyRuleDTO(r))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (a *api) updatePolicyRule(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	policyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	ruleID, err := uuid.Parse(r.PathValue("ruleId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid rule ID"))
		return
	}
	
	var in struct {
		Description *string         `json:"description"`
		Conditions  *map[string]any `json:"conditions"`
		Parameters  *map[string]any `json:"parameters"`
		Enabled     *bool           `json:"enabled"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	
	update := domain.UpdatePolicyRule{
		Description: in.Description,
		Enabled:     in.Enabled,
	}
	if in.Conditions != nil {
		update.Conditions, _ = json.Marshal(*in.Conditions)
	}
	if in.Parameters != nil {
		update.Parameters, _ = json.Marshal(*in.Parameters)
	}

	rule, err := a.policy.UpdateRule(r.Context(), actor, policyID, ruleID, update)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPolicyRuleDTO(rule))
}

func (a *api) deletePolicyRule(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	policyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}
	ruleID, err := uuid.Parse(r.PathValue("ruleId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid rule ID"))
		return
	}
	if err := a.policy.DeleteRule(r.Context(), actor, policyID, ruleID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) previewPolicy(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	var in struct {
		Model            string     `json:"model"`
		Provider         string     `json:"provider"`
		InputTokens      int        `json:"input_tokens"`
		OutputTokens     int        `json:"output_tokens"`
		TeamID           *uuid.UUID `json:"team_id"`
		ProjectID        *uuid.UUID `json:"project_id"`
		UserID           *uuid.UUID `json:"user_id"`
		ServiceAccountID *uuid.UUID `json:"service_account_id"`
		CredentialID     *uuid.UUID `json:"credential_id"`
		Scopes           []string   `json:"scopes"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}

	if in.Model == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("model is required for preview"))
		return
	}

	policies, err := a.policy.ListPolicies(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	// Collect rules for active policies
	var activeRules []domain.PolicyRule
	for _, pol := range policies {
		if pol.Enforcement != domain.EnforcementActive {
			continue
		}
		rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, pol.ID)
		if err == nil {
			for _, r := range rules {
				if r.Enabled {
					activeRules = append(activeRules, r)
				}
			}
		}
	}

	// Evaluate scopes from highest specificity to lowest:
	// Level 5: credential
	// Level 4: user / service_account
	// Level 3: project
	// Level 2: team
	// Level 1: organization
	decision := "allow"
	reason := ""
	effectiveScope := "organization"
	var matchedRules []policyRuleDTO

	// Check scopes
	credIDStr := ""
	if in.CredentialID != nil {
		credIDStr = in.CredentialID.String()
	}
	teamIDStr := ""
	if in.TeamID != nil {
		teamIDStr = in.TeamID.String()
	}
	projIDStr := ""
	if in.ProjectID != nil {
		projIDStr = in.ProjectID.String()
	}
	userIDStr := ""
	if in.UserID != nil {
		userIDStr = in.UserID.String()
	}
	saIDStr := ""
	if in.ServiceAccountID != nil {
		saIDStr = in.ServiceAccountID.String()
	}

	matchRuleScope := func(r domain.PolicyRule) (bool, int) {
		var cond map[string]any
		_ = json.Unmarshal(r.Conditions, &cond)
		var param map[string]any
		_ = json.Unmarshal(r.Parameters, &param)

		scope := toStringVal(cond["scope"])
		if scope == "" {
			scope = toStringVal(param["scope"])
		}
		scopeID := toStringVal(cond["scope_id"])
		if scopeID == "" {
			scopeID = toStringVal(param["scope_id"])
		}

		switch scope {
		case "credential", "gateway_credential":
			if credIDStr != "" && (scopeID == "" || scopeID == credIDStr) {
				return true, 5
			}
			return false, 5
		case "user":
			if userIDStr != "" && (scopeID == "" || scopeID == userIDStr) {
				return true, 4
			}
			return false, 4
		case "service_account":
			if saIDStr != "" && (scopeID == "" || scopeID == saIDStr) {
				return true, 4
			}
			return false, 4
		case "project":
			if projIDStr != "" && (scopeID == "" || scopeID == projIDStr) {
				return true, 3
			}
			return false, 3
		case "team":
			if teamIDStr != "" && (scopeID == "" || scopeID == teamIDStr) {
				return true, 2
			}
			return false, 2
		case "organization", "org", "":
			return true, 1
		default:
			return true, 1
		}
	}

	for rank := 5; rank >= 1; rank-- {
		var rankRules []domain.PolicyRule
		for _, r := range activeRules {
			if r.RuleType != domain.RuleTypeAllowlist && r.RuleType != domain.RuleTypeBlocklist {
				continue
			}
			matched, rRank := matchRuleScope(r)
			if matched && rRank == rank {
				rankRules = append(rankRules, r)
			}
		}
		if len(rankRules) == 0 {
			continue
		}

		// Check Blocklists first (Deny-overrides)
		for _, r := range rankRules {
			if r.RuleType == domain.RuleTypeBlocklist && ruleMatchesModelStr(r, in.Model) {
				decision = "deny"
				reason = "model \"" + in.Model + "\" is blocked by policy"
				effectiveScope = rankToScopeName(rank)
				matchedRules = append(matchedRules, toPolicyRuleDTO(r))
				break
			}
		}
		if decision == "deny" {
			break
		}

		// Check Allowlists
		var allowlists []domain.PolicyRule
		for _, r := range rankRules {
			if r.RuleType == domain.RuleTypeAllowlist {
				allowlists = append(allowlists, r)
			}
		}
		if len(allowlists) > 0 {
			allowedByList := false
			for _, r := range allowlists {
				if ruleMatchesModelStr(r, in.Model) {
					allowedByList = true
					matchedRules = append(matchedRules, toPolicyRuleDTO(r))
					break
				}
			}
			if allowedByList {
				decision = "allow"
				effectiveScope = rankToScopeName(rank)
				break
			} else {
				decision = "deny"
				reason = "model \"" + in.Model + "\" is not allowed by policy"
				effectiveScope = rankToScopeName(rank)
				for _, r := range allowlists {
					matchedRules = append(matchedRules, toPolicyRuleDTO(r))
				}
				break
			}
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"allowed":         decision == "allow",
		"decision":        decision,
		"reason":          reason,
		"effective_scope": effectiveScope,
		"matched_rules":   matchedRules,
	})
}

func (a *api) validatePolicy(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid policy ID"))
		return
	}

	pol, err := a.policy.GetPolicy(r.Context(), actor.OrganizationID, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	var validationErrors []string
	var warnings []string

	if pol.Name == "" {
		validationErrors = append(validationErrors, "policy name cannot be empty")
	}
	if pol.Enforcement != "active" && pol.Enforcement != "permissive" && pol.Enforcement != "disabled" {
		validationErrors = append(validationErrors, "enforcement must be 'active', 'permissive', or 'disabled'")
	}

	validRuleTypes := map[string]bool{
		domain.RuleTypeAllowlist:     true,
		domain.RuleTypeBlocklist:     true,
		domain.RuleTypeRateLimit:     true,
		domain.RuleTypeTokenBudget:   true,
		domain.RuleTypeBudget:        true,
		domain.RuleTypeContentFilter: true,
	}

	allowlistModels := make(map[string]map[string]bool)
	blocklistModels := make(map[string]map[string]bool)

	for _, rule := range rules {
		if !validRuleTypes[rule.RuleType] {
			validationErrors = append(validationErrors, "rule "+rule.ID.String()+" has unsupported type: "+rule.RuleType)
		}

		var cond map[string]any
		if len(rule.Conditions) > 0 {
			if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
				validationErrors = append(validationErrors, "rule "+rule.ID.String()+" has malformed conditions JSON")
			}
		}
		var param map[string]any
		if len(rule.Parameters) > 0 {
			if err := json.Unmarshal(rule.Parameters, &param); err != nil {
				validationErrors = append(validationErrors, "rule "+rule.ID.String()+" has malformed parameters JSON")
			}
		}

		scope := toStringVal(cond["scope"])
		if scope == "" {
			scope = toStringVal(param["scope"])
		}
		if scope == "" {
			scope = "organization"
		}

		scopeID := toStringVal(cond["scope_id"])
		if scopeID == "" {
			scopeID = toStringVal(param["scope_id"])
		}
		if scopeID != "" {
			if _, err := uuid.Parse(scopeID); err != nil {
				validationErrors = append(validationErrors, "rule "+rule.ID.String()+" has invalid scope_id: "+scopeID)
			}
		}

		switch rule.RuleType {
		case domain.RuleTypeAllowlist, domain.RuleTypeBlocklist:
			models := extractModelsFromMap(param, cond)
			if len(models) == 0 {
				warnings = append(warnings, "rule "+rule.ID.String()+" does not specify any models")
			}
			scopeKey := scope
			if scopeID != "" {
				scopeKey = scope + ":" + scopeID
			}
			for _, m := range models {
				if rule.RuleType == domain.RuleTypeAllowlist {
					if allowlistModels[scopeKey] == nil {
						allowlistModels[scopeKey] = make(map[string]bool)
					}
					allowlistModels[scopeKey][m] = true
				} else {
					if blocklistModels[scopeKey] == nil {
						blocklistModels[scopeKey] = make(map[string]bool)
					}
					blocklistModels[scopeKey][m] = true
				}
			}

		case domain.RuleTypeRateLimit:
			limit := toNumeric(param["limit"], cond["limit"])
			if limit <= 0 {
				validationErrors = append(validationErrors, "rate limit rule "+rule.ID.String()+" must specify limit > 0")
			}
		case domain.RuleTypeBudget, domain.RuleTypeTokenBudget:
			limit := toNumeric(param["limit"], cond["limit"])
			if limit <= 0 {
				validationErrors = append(validationErrors, "budget rule "+rule.ID.String()+" must specify limit > 0")
			}
		}
	}

	// Conflict detection
	for scopeKey, ams := range allowlistModels {
		if bms, ok := blocklistModels[scopeKey]; ok {
			for m := range ams {
				if bms[m] {
					warnings = append(warnings, "conflict in scope "+scopeKey+": model \""+m+"\" is in both allowlist and blocklist (deny-overrides applies)")
				}
			}
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"valid":       len(validationErrors) == 0,
		"policy_id":   pol.ID.String(),
		"name":        pol.Name,
		"enforcement": pol.Enforcement,
		"rules_count": len(rules),
		"errors":      validationErrors,
		"warnings":    warnings,
	})
}

func toStringVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toNumeric(vals ...any) float64 {
	for _, v := range vals {
		if v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			if n > 0 {
				return n
			}
		case int:
			if n > 0 {
				return float64(n)
			}
		case json.Number:
			if f, err := n.Float64(); err == nil && f > 0 {
				return f
			}
		}
	}
	return 0
}

func extractModelsFromMap(param, cond map[string]any) []string {
	var models []string
	extract := func(m map[string]any) {
		if m == nil {
			return
		}
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
	}
	extract(param)
	if len(models) == 0 {
		extract(cond)
	}
	return models
}

func ruleMatchesModelStr(r domain.PolicyRule, model string) bool {
	var cond, param map[string]any
	_ = json.Unmarshal(r.Conditions, &cond)
	_ = json.Unmarshal(r.Parameters, &param)
	models := extractModelsFromMap(param, cond)
	for _, entry := range models {
		if entry == "*" || entry == model {
			return true
		}
		if strings.HasSuffix(entry, "/"+model) || strings.HasSuffix(model, "/"+entry) {
			return true
		}
		if strings.HasSuffix(entry, "/*") {
			prefix := strings.TrimSuffix(entry, "/*")
			if strings.HasPrefix(model, prefix+"/") {
				return true
			}
		}
		if strings.HasSuffix(entry, "*") {
			prefix := strings.TrimSuffix(entry, "*")
			if strings.HasPrefix(model, prefix) || strings.Contains(model, "/"+prefix) {
				return true
			}
		}
	}
	return false
}

func rankToScopeName(rank int) string {
	switch rank {
	case 5:
		return "credential"
	case 4:
		return "user_or_service_account"
	case 3:
		return "project"
	case 2:
		return "team"
	default:
		return "organization"
	}
}
