package controlapi

import (
	"encoding/json"
	"errors"
	"net/http"
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
	// TODO: implement preview
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "preview not implemented"})
}

func (a *api) validatePolicy(w http.ResponseWriter, r *http.Request) {
	// TODO: implement validation
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": true})
}
