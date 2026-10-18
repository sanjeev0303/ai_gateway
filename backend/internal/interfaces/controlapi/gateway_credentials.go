package controlapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- gateway credential DTOs ------------------------------------------------

type gatewayCredentialDTO struct {
	ID                string   `json:"id"`
	OrganizationID    string   `json:"organization_id"`
	Name              string   `json:"name"`
	KeyPrefix         string   `json:"key_prefix"`
	SubjectType       string   `json:"subject_type"`
	SubjectID         string   `json:"subject_id"`
	TeamID            *string  `json:"team_id,omitempty"`
	ProjectID         *string  `json:"project_id,omitempty"`
	Scopes            []string `json:"scopes"`
	ProviderAccountID *string  `json:"provider_account_id,omitempty"`
	AllowedProviders  []string `json:"allowed_providers,omitempty"`
	AllowedModels     []string `json:"allowed_models,omitempty"`
	RateLimit         *int     `json:"rate_limit,omitempty"`
	BudgetLimit       *float64 `json:"budget_limit,omitempty"`
	Status            string   `json:"status"`
	ExpiresAt         *string  `json:"expires_at,omitempty"`
	LastUsedAt        *string  `json:"last_used_at,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

func toGatewayCredentialDTO(c domain.GatewayCredential) gatewayCredentialDTO {
	scopes := make([]string, len(c.Scopes))
	for i, s := range c.Scopes {
		scopes[i] = string(s)
	}
	var teamID, projectID, providerAccountID *string
	if c.TeamID != nil {
		s := c.TeamID.String()
		teamID = &s
	}
	if c.ProjectID != nil {
		s := c.ProjectID.String()
		projectID = &s
	}
	if c.ProviderAccountID != nil {
		s := c.ProviderAccountID.String()
		providerAccountID = &s
	}

	allowedProviders := make([]string, len(c.AllowedProviders))
	copy(allowedProviders, c.AllowedProviders)

	allowedModels := make([]string, len(c.AllowedModels))
	copy(allowedModels, c.AllowedModels)

	return gatewayCredentialDTO{
		ID:                c.ID.String(),
		OrganizationID:    c.OrganizationID.String(),
		Name:              c.Name,
		KeyPrefix:         c.KeyPrefix,
		SubjectType:       string(c.SubjectType),
		SubjectID:         c.SubjectID.String(),
		TeamID:            teamID,
		ProjectID:         projectID,
		Scopes:            scopes,
		ProviderAccountID: providerAccountID,
		AllowedProviders:  allowedProviders,
		AllowedModels:     allowedModels,
		RateLimit:         c.RateLimit,
		BudgetLimit:       c.BudgetLimit,
		Status:            string(c.Status),
		ExpiresAt:         toTimePtr(c.ExpiresAt),
		LastUsedAt:        toTimePtr(c.LastUsedAt),
		CreatedAt:         ts(c.CreatedAt),
		UpdatedAt:         ts(c.UpdatedAt),
	}
}

func toTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// ---- request bodies ---------------------------------------------------------

type createGatewayCredentialRequest struct {
	Name              string   `json:"name"`
	SubjectType       string   `json:"subject_type"`
	SubjectID         string   `json:"subject_id"`
	TeamID            *string  `json:"team_id,omitempty"`
	ProjectID         *string  `json:"project_id,omitempty"`
	Scopes            []string `json:"scopes"`
	ProviderAccountID *string  `json:"provider_account_id,omitempty"`
	AllowedProviders  []string `json:"allowed_providers,omitempty"`
	AllowedModels     []string `json:"allowed_models,omitempty"`
	RateLimit         *int     `json:"rate_limit,omitempty"`
	BudgetLimit       *float64 `json:"budget_limit,omitempty"`
	ExpiresAt         *string  `json:"expires_at,omitempty"`
}

type rotateGatewayCredentialRequest struct {
	Name              *string  `json:"name,omitempty"`
	Scopes            []string `json:"scopes,omitempty"`
	ProviderAccountID *string  `json:"provider_account_id,omitempty"`
	AllowedProviders  []string `json:"allowed_providers,omitempty"`
	AllowedModels     []string `json:"allowed_models,omitempty"`
	RateLimit         *int     `json:"rate_limit,omitempty"`
	BudgetLimit       *float64 `json:"budget_limit,omitempty"`
	ExpiresAt         *string  `json:"expires_at,omitempty"`
}

// ---- handlers ---------------------------------------------------------------

func (a *api) createGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createGatewayCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	subjectType := domain.SubjectType(req.SubjectType)
	if !domain.IsValidSubjectType(subjectType) {
		writeServiceError(w, r, repositories.ErrInvalid)
		return
	}
	scopes := make([]domain.GatewayCredentialScope, len(req.Scopes))
	for i, s := range req.Scopes {
		scopes[i] = domain.GatewayCredentialScope(s)
	}
	for _, scope := range scopes {
		if !domain.IsValidGatewayCredentialScope(scope) {
			writeServiceError(w, r, repositories.ErrInvalid)
			return
		}
	}
	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var teamID, projectID *uuid.UUID
	if req.TeamID != nil {
		id, err := uuid.Parse(*req.TeamID)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		teamID = &id
	}
	if req.ProjectID != nil {
		id, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		projectID = &id
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		expiresAt = &t
	}

	var providerAccountID *uuid.UUID
	if req.ProviderAccountID != nil {
		id, err := uuid.Parse(*req.ProviderAccountID)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		providerAccountID = &id
	}

	credential, rawKey, err := a.svc.CreateGatewayCredential(r.Context(), actor, domain.NewGatewayCredential{
		Name:              req.Name,
		SubjectType:       subjectType,
		SubjectID:         subjectID,
		TeamID:            teamID,
		ProjectID:         projectID,
		Scopes:            scopes,
		ProviderAccountID: providerAccountID,
		AllowedProviders:  req.AllowedProviders,
		AllowedModels:     req.AllowedModels,
		RateLimit:         req.RateLimit,
		BudgetLimit:       req.BudgetLimit,
		ExpiresAt:         expiresAt,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	dto := toGatewayCredentialDTO(credential)
	dto.KeyPrefix = rawKey
	httpx.WriteJSON(w, http.StatusCreated, dto)
}

func (a *api) listGatewayCredentials(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	credentials, err := a.svc.ListGatewayCredentials(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]gatewayCredentialDTO, 0, len(credentials))
	for _, c := range credentials {
		items = append(items, toGatewayCredentialDTO(c))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) getGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	credential, err := a.svc.GetGatewayCredential(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGatewayCredentialDTO(credential))
}

func (a *api) revokeGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RevokeGatewayCredential(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) rotateGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req rotateGatewayCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}

	input := domain.NewGatewayCredential{}
	if req.Name != nil {
		input.Name = *req.Name
	}
	if len(req.Scopes) > 0 {
		scopes := make([]domain.GatewayCredentialScope, len(req.Scopes))
		for i, s := range req.Scopes {
			scopes[i] = domain.GatewayCredentialScope(s)
		}
		input.Scopes = scopes
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		input.ExpiresAt = &t
	}
	if req.ProviderAccountID != nil {
		id, err := uuid.Parse(*req.ProviderAccountID)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		input.ProviderAccountID = &id
	}
	if len(req.AllowedProviders) > 0 {
		input.AllowedProviders = req.AllowedProviders
	}
	if len(req.AllowedModels) > 0 {
		input.AllowedModels = req.AllowedModels
	}
	if req.RateLimit != nil {
		input.RateLimit = req.RateLimit
	}
	if req.BudgetLimit != nil {
		input.BudgetLimit = req.BudgetLimit
	}

	rotated, rawKey, err := a.svc.RotateGatewayCredential(r.Context(), actor, id, input)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	dto := toGatewayCredentialDTO(rotated)
	dto.KeyPrefix = rawKey
	httpx.WriteJSON(w, http.StatusCreated, dto)
}

func (a *api) expireGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.ExpireGatewayCredential(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) restoreGatewayCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RestoreGatewayCredential(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
