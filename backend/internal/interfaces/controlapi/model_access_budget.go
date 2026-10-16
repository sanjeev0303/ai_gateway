package controlapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/platform/httpx"

	"github.com/google/uuid"
)

type budgetDTO struct {
	ID              string  `json:"id"`
	OrganizationID  string  `json:"organization_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Scope           string  `json:"scope"`
	ScopeID         string  `json:"scope_id"`
	LimitType       string  `json:"limit_type"`
	LimitAmount     float64 `json:"limit_amount"`
	Window          string  `json:"window"`
	AlertThreshold  float64 `json:"alert_threshold"`
	Enabled         bool    `json:"enabled"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	CurrentSpend    float64 `json:"current_spend"`
	CurrentTokens   int64   `json:"current_tokens"`
	CurrentRequests int64   `json:"current_requests"`
	WindowStart     string  `json:"window_start"`
	WindowEnd       string  `json:"window_end"`
	PercentageUsed  float64 `json:"percentage_used"`
}

type modelAccessEntryDTO struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Allowed   bool   `json:"allowed"`
	Source    string `json:"source"`
	PolicyID  string `json:"policy_id"`
	RuleID    string `json:"rule_id"`
	Priority  int32  `json:"priority"`
	Effective bool   `json:"effective"`
}

type modelAccessPreviewRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Action   string `json:"action"`
}

type modelAccessPreviewResponse struct {
	Provider          string               `json:"provider"`
	Model             string               `json:"model"`
	CurrentDecision   string               `json:"current_decision"`
	PreviewDecision   string               `json:"preview_decision"`
	WouldChange       bool                 `json:"would_change"`
	BlockingRules     []modelAccessEntryDTO `json:"blocking_rules"`
	AllowingRules     []modelAccessEntryDTO `json:"allowing_rules"`
}

func toBudgetDTO(b domain.Budget) budgetDTO {
	pct := 0.0
	if b.LimitAmount > 0 {
		if b.LimitType == domain.BudgetLimitTypeTokens {
			pct = (float64(b.CurrentTokens) / b.LimitAmount) * 100
		} else if b.LimitType == domain.BudgetLimitTypeRequests {
			pct = (float64(b.CurrentRequests) / b.LimitAmount) * 100
		} else {
			pct = (b.CurrentSpend / b.LimitAmount) * 100
		}
	}
	return budgetDTO{
		ID:              b.ID.String(),
		OrganizationID:  b.OrganizationID.String(),
		Name:            b.Name,
		Description:     b.Description,
		Scope:           string(b.Scope),
		ScopeID:         b.ScopeID.String(),
		LimitType:       string(b.LimitType),
		LimitAmount:     b.LimitAmount,
		Window:          string(b.Window),
		AlertThreshold:  b.AlertThreshold,
		Enabled:         b.Enabled,
		CreatedAt:       b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       b.UpdatedAt.Format(time.RFC3339),
		CurrentSpend:    b.CurrentSpend,
		CurrentTokens:   b.CurrentTokens,
		CurrentRequests: b.CurrentRequests,
		WindowStart:     b.WindowStart.Format(time.RFC3339),
		WindowEnd:       b.WindowEnd.Format(time.RFC3339),
		PercentageUsed:  pct,
	}
}

func toModelAccessEntryDTO(e domain.ModelAccessEntry) modelAccessEntryDTO {
	return modelAccessEntryDTO{
		Provider:  e.Provider,
		Model:     e.Model,
		Allowed:   e.Allowed,
		Source:    e.Source,
		PolicyID:  e.PolicyID.String(),
		RuleID:    e.RuleID.String(),
		Priority:  e.Priority,
		Effective: e.Effective,
	}
}

func (a *api) listModelAccess(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	// Get all policies and rules to build model access matrix
	policies, err := a.policy.ListPolicies(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	var entries []modelAccessEntryDTO
	for _, pol := range policies {
		rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, pol.ID)
		if err != nil {
			continue
		}
		for _, rule := range rules {
			if rule.RuleType != domain.RuleTypeAllowlist && rule.RuleType != domain.RuleTypeBlocklist {
				continue
			}
			var cond map[string]any
			if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
				continue
			}
			provider, _ := cond["provider"].(string)
			model, _ := cond["model"].(string)
			if provider == "" || model == "" {
				continue
			}
			entries = append(entries, modelAccessEntryDTO{
				Provider:  provider,
				Model:     model,
				Allowed:   rule.RuleType == domain.RuleTypeAllowlist,
				Source:    rule.RuleType,
				PolicyID:  pol.ID.String(),
				RuleID:    rule.ID.String(),
				Priority:  pol.Priority,
				Effective: true,
			})
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": entries})
}

func (a *api) updateModelAccess(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Role != domain.RoleOwner {
		writeServiceError(w, r, repositories.ErrForbidden)
		return
	}

	var in struct {
		Entries []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Action   string `json:"action"` // "allow" or "block"
		} `json:"entries"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}

	// Find or create a model access policy
	pol, err := a.findOrCreateModelAccessPolicy(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	// Process each entry
	for _, entry := range in.Entries {
		ruleType := domain.RuleTypeBlocklist
		if entry.Action == "allow" {
			ruleType = domain.RuleTypeAllowlist
		}

		condBytes, _ := json.Marshal(map[string]any{
			"provider": entry.Provider,
			"model":    entry.Model,
		})
		paramBytes, _ := json.Marshal(map[string]any{})

		// Check if rule exists
		rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, pol.ID)
		if err != nil {
			continue
		}

		found := false
		for _, rule := range rules {
			var cond map[string]any
			if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
				continue
			}
			if cond["provider"] == entry.Provider && cond["model"] == entry.Model {
				// Update existing rule
				update := domain.UpdatePolicyRule{
					Enabled:     ptr(true),
					Conditions:  condBytes,
					Parameters:  paramBytes,
				}
				_, _ = a.policy.UpdateRule(r.Context(), actor, pol.ID, rule.ID, update)
				found = true
				break
			}
		}

		if !found {
			// Create new rule
			_, _ = a.policy.CreateRule(r.Context(), actor, domain.NewPolicyRule{
				PolicyID:    pol.ID,
				RuleType:    ruleType,
				Description: "Model access: " + entry.Provider + "/" + entry.Model,
				Conditions:  condBytes,
				Parameters:  paramBytes,
				Enabled:     true,
			})
		}
	}

	// Policy invalidation is handled by the service

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

func (a *api) previewModelAccess(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Role != domain.RoleOwner {
		writeServiceError(w, r, repositories.ErrForbidden)
		return
	}

	var in modelAccessPreviewRequest
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}

	// Get current decision
	policies, err := a.policy.ListPolicies(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	var currentDecision string
	var blockingRules []modelAccessEntryDTO
	var allowingRules []modelAccessEntryDTO

	for _, pol := range policies {
		rules, err := a.policy.ListRules(r.Context(), actor.OrganizationID, pol.ID)
		if err != nil {
			continue
		}
		for _, rule := range rules {
			if rule.RuleType != domain.RuleTypeAllowlist && rule.RuleType != domain.RuleTypeBlocklist {
				continue
			}
			var cond map[string]any
			if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
				continue
			}
			provider, _ := cond["provider"].(string)
			model, _ := cond["model"].(string)
			if provider != in.Provider || model != in.Model {
				continue
			}
			entry := toModelAccessEntryDTO(domain.ModelAccessEntry{
				Provider: provider,
				Model:    model,
				Allowed:  rule.RuleType == domain.RuleTypeAllowlist,
				Source:   rule.RuleType,
				PolicyID: pol.ID,
				RuleID:   rule.ID,
				Priority: pol.Priority,
			})
			if rule.RuleType == domain.RuleTypeBlocklist {
				blockingRules = append(blockingRules, entry)
				if currentDecision == "" {
					currentDecision = "block"
				}
			} else {
				allowingRules = append(allowingRules, entry)
				if currentDecision == "" {
					currentDecision = "allow"
				}
			}
		}
	}

	if currentDecision == "" {
		currentDecision = "allow" // default allow
	}

	// Preview decision
	previewDecision := currentDecision
	wouldChange := false
	if in.Action == "allow" && currentDecision == "block" {
		previewDecision = "allow"
		wouldChange = true
	} else if in.Action == "block" && currentDecision == "allow" {
		previewDecision = "block"
		wouldChange = true
	}

	httpx.WriteJSON(w, http.StatusOK, modelAccessPreviewResponse{
		Provider:        in.Provider,
		Model:           in.Model,
		CurrentDecision: currentDecision,
		PreviewDecision: previewDecision,
		WouldChange:     wouldChange,
		BlockingRules:   blockingRules,
		AllowingRules:   allowingRules,
	})
}

func (a *api) findOrCreateModelAccessPolicy(ctx context.Context, orgID uuid.UUID) (domain.Policy, error) {
	policies, err := a.policy.ListPolicies(ctx, orgID)
	if err != nil {
		return domain.Policy{}, err
	}
	for _, p := range policies {
		if p.Name == "Model Access Control" {
			return p, nil
		}
	}
	return a.policy.CreatePolicy(ctx, domain.Actor{OrganizationID: orgID}, domain.NewPolicy{
		Name:        "Model Access Control",
		Description: "Auto-generated policy for model allowlist/blocklist rules",
		Enforcement: domain.EnforcementActive,
		Priority:    50,
	})
}

func (a *api) listBudgets(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	filter := repositories.BudgetFilter{}
	if scope := r.URL.Query().Get("scope"); scope != "" {
		s := domain.BudgetScope(scope)
		filter.Scope = &s
	}
	if scopeID := r.URL.Query().Get("scope_id"); scopeID != "" {
		if id, err := uuid.Parse(scopeID); err == nil {
			filter.ScopeID = &id
		}
	}
	if limitType := r.URL.Query().Get("limit_type"); limitType != "" {
		lt := domain.BudgetLimitType(limitType)
		filter.LimitType = &lt
	}
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		e := enabled == "true"
		filter.Enabled = &e
	}

	budgets, err := a.budget.List(r.Context(), actor.OrganizationID, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	out := make([]budgetDTO, 0, len(budgets))
	for _, b := range budgets {
		windowStart := startOfMonth
		windowEnd := endOfMonth
		if b.Window == domain.BudgetWindowDaily {
			windowStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			windowEnd = windowStart.AddDate(0, 0, 1)
		} else if b.Window == domain.BudgetWindowWeekly {
			windowStart = now.AddDate(0, 0, -7)
			windowEnd = now
		}

		if a.usage != nil {
			usageFilter := domain.UsageFilter{
				From: windowStart,
				To:   windowEnd,
			}
			if b.Scope == domain.BudgetScopeProject && b.ScopeID != uuid.Nil {
				usageFilter.ProjectID = &b.ScopeID
			}

			if summary, err := a.usage.GetSummary(r.Context(), actor, usageFilter); err == nil && summary.Requests > 0 {
				b.CurrentSpend = float64(summary.TotalCostMicros) / 1000000.0
				b.CurrentTokens = summary.TotalTokens
				b.CurrentRequests = summary.Requests
			}
		}
		if b.WindowStart.IsZero() {
			b.WindowStart = windowStart
		}
		if b.WindowEnd.IsZero() {
			b.WindowEnd = windowEnd
		}

		out = append(out, toBudgetDTO(b))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (a *api) createBudget(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Role != domain.RoleOwner {
		writeServiceError(w, r, repositories.ErrForbidden)
		return
	}

	var in struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		Scope          string  `json:"scope"`
		ScopeID        string  `json:"scope_id"`
		LimitType      string  `json:"limit_type"`
		LimitAmount    float64 `json:"limit_amount"`
		Window         string  `json:"window"`
		AlertThreshold float64 `json:"alert_threshold"`
		Enabled        bool    `json:"enabled"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}

	scopeID := uuid.Nil
	if in.ScopeID != "" {
		var err error
		scopeID, err = uuid.Parse(in.ScopeID)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid scope_id"))
			return
		}
	}

	b, err := a.budget.Create(r.Context(), actor, domain.NewBudget{
		Name:            in.Name,
		Description:     in.Description,
		Scope:           domain.BudgetScope(in.Scope),
		ScopeID:         scopeID,
		LimitType:       domain.BudgetLimitType(in.LimitType),
		LimitAmount:     in.LimitAmount,
		Window:          domain.BudgetWindow(in.Window),
		AlertThreshold:  in.AlertThreshold,
		Enabled:         in.Enabled,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toBudgetDTO(b))
}

func (a *api) getBudget(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid budget ID"))
		return
	}
	b, err := a.budget.Get(r.Context(), actor.OrganizationID, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBudgetDTO(b))
}

func (a *api) updateBudget(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Role != domain.RoleOwner {
		writeServiceError(w, r, repositories.ErrForbidden)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid budget ID"))
		return
	}
	var in struct {
		Name           *string  `json:"name"`
		Description    *string  `json:"description"`
		LimitAmount    *float64 `json:"limit_amount"`
		AlertThreshold *float64 `json:"alert_threshold"`
		Enabled        *bool    `json:"enabled"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	b, err := a.budget.Update(r.Context(), actor, id, domain.UpdateBudget{
		Name:           in.Name,
		Description:    in.Description,
		LimitAmount:    in.LimitAmount,
		AlertThreshold: in.AlertThreshold,
		Enabled:        in.Enabled,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBudgetDTO(b))
}

func (a *api) deleteBudget(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Role != domain.RoleOwner {
		writeServiceError(w, r, repositories.ErrForbidden)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.RequestIDFrom(r.Context()), errors.New("invalid budget ID"))
		return
	}
	if err := a.budget.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ptr[T any](v T) *T {
	return &v
}