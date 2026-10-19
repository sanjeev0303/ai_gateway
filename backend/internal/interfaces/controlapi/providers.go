package controlapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ---- provider DTOs --------------------------------------------------------

type providerAccountDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ProviderKind   string `json:"provider_kind"`
	Name           string `json:"name"`
	BaseURL        string `json:"base_url,omitempty"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// providerCredentialDTO never exposes the SecretRef or the stored secret; the
// value is reachable only through the gateway proxy, and identification is
// limited to the fingerprint and the last-four tail.
type providerCredentialDTO struct {
	ID                string `json:"id"`
	OrganizationID    string `json:"organization_id"`
	ProviderAccountID string `json:"provider_account_id"`
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	Fingerprint       string `json:"fingerprint"`
	Tail              string `json:"tail"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type modelDTO struct {
	ID                string   `json:"id"`
	ProviderKind      string   `json:"provider_kind"`
	ModelSlug         string   `json:"model_slug"`
	DisplayName       string   `json:"display_name"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ContextLimit      int      `json:"context_limit,omitempty"`
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsReasoning bool     `json:"supports_reasoning"`
	Status            string   `json:"status,omitempty"`
	PricingVersion    string   `json:"pricing_version,omitempty"`
	IsActive          bool     `json:"is_active"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

func toProviderAccountDTO(a domain.ProviderAccount) providerAccountDTO {
	return providerAccountDTO{
		ID:             a.ID.String(),
		OrganizationID: a.OrganizationID.String(),
		ProviderKind:   a.ProviderKind,
		Name:           a.Name,
		BaseURL:        a.BaseURL,
		Status:         a.Status,
		CreatedAt:      ts(a.CreatedAt),
		UpdatedAt:      ts(a.UpdatedAt),
	}
}

func toProviderCredentialDTO(c domain.ProviderCredential) providerCredentialDTO {
	return providerCredentialDTO{
		ID:                c.ID.String(),
		OrganizationID:    c.OrganizationID.String(),
		ProviderAccountID: c.ProviderAccountID.String(),
		Name:              c.Name,
		Kind:              c.Kind,
		Fingerprint:       c.Fingerprint,
		Tail:              c.Tail,
		Status:            c.Status,
		CreatedAt:         ts(c.CreatedAt),
		UpdatedAt:         ts(c.UpdatedAt),
	}
}

func toModelDTO(m domain.Model) modelDTO {
	status := m.Status
	if status == "" {
		if m.IsActive {
			status = "active"
		} else {
			status = "inactive"
		}
	}
	return modelDTO{
		ID:                m.ID.String(),
		ProviderKind:      m.ProviderKind,
		ModelSlug:         m.ModelSlug,
		DisplayName:       m.DisplayName,
		Capabilities:      m.Capabilities,
		ContextLimit:      m.ContextLimit,
		SupportsStreaming: m.SupportsStreaming,
		SupportsTools:     m.SupportsTools,
		SupportsReasoning: m.SupportsReasoning,
		Status:            status,
		PricingVersion:    m.PricingVersion,
		IsActive:          m.IsActive,
		CreatedAt:         ts(m.CreatedAt),
		UpdatedAt:         ts(m.UpdatedAt),
	}
}

// ---- request bodies -------------------------------------------------------

type createProviderAccountRequest struct {
	ProviderKind string `json:"provider_kind"`
	Name         string `json:"name"`
	BaseURL      string `json:"base_url,omitempty"`
}

type updateProviderAccountRequest struct {
	Status string `json:"status"`
}

type createProviderCredentialRequest struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Secret string `json:"secret"`
}

type rotateProviderCredentialRequest struct {
	Secret string `json:"secret"`
}

// providerProbeRequest carries a secret for a live probe. The secret is used
// for the upstream call only: it is never stored, never returned, and never
// logged. A caller that wants the secret persisted should create a credential.
type providerProbeRequest struct {
	ProviderKind string `json:"provider_kind"`
	APIKey       string `json:"api_key"`
}

type providerHealthDTO struct {
	ProviderKind string `json:"provider_kind"`
	Status       string `json:"status"`
	StatusReason string `json:"status_reason,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	CheckedAt    string `json:"checked_at"`
}

func toProviderHealthDTO(s domain.ProviderHealthState) providerHealthDTO {
	return providerHealthDTO{
		ProviderKind: s.ProviderKind,
		Status:       s.Status,
		StatusReason: s.StatusReason,
		LastError:    s.LastError,
		CheckedAt:    s.CheckedAt.UTC().Format(time.RFC3339),
	}
}

// ---- provider account handlers -------------------------------------------

func (a *api) createProviderAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createProviderAccountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	account, err := a.providers.CreateAccount(r.Context(), actor, domain.NewProviderAccount{
		ProviderKind: req.ProviderKind,
		Name:         req.Name,
		BaseURL:      req.BaseURL,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProviderAccountDTO(account))
}

func (a *api) listProviderAccounts(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	accounts, err := a.providers.ListAccounts(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]providerAccountDTO, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, toProviderAccountDTO(account))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) getProviderAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	account, err := a.providers.GetAccount(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProviderAccountDTO(account))
}

func (a *api) updateProviderAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req updateProviderAccountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	account, err := a.providers.UpdateAccountStatus(r.Context(), actor, id, req.Status)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProviderAccountDTO(account))
}

func (a *api) deleteProviderAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.providers.DeleteAccount(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) validateAccountCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	err = a.providers.ValidateAccount(r.Context(), actor, id)
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": true})
		return
	}
	if errors.Is(err, provider.ErrInvalidProviderCredential) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}
	writeServiceError(w, r, err)
}

func (a *api) syncAccountModels(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	models, err := a.providers.SyncAccountModels(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]modelDTO, 0, len(models))
	for _, m := range models {
		items = append(items, toModelDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) checkAccountHealth(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	state, err := a.providers.CheckAccountHealth(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProviderHealthDTO(state))
}

// ---- provider credential handlers ----------------------------------------

func (a *api) createProviderCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	accountID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req createProviderCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	credential, err := a.providers.CreateCredential(r.Context(), actor, accountID, req.Name, req.Kind, req.Secret)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProviderCredentialDTO(credential))
}

func (a *api) listProviderCredentials(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	credentials, err := a.providers.ListCredentials(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]providerCredentialDTO, 0, len(credentials))
	for _, credential := range credentials {
		items = append(items, toProviderCredentialDTO(credential))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) getProviderCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	credential, err := a.providers.GetCredential(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProviderCredentialDTO(credential))
}

func (a *api) revokeProviderCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.providers.RevokeCredential(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) rotateProviderCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req rotateProviderCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	rotated, err := a.providers.RotateCredential(r.Context(), actor, id, req.Secret)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProviderCredentialDTO(rotated))
}

// ---- model handlers -------------------------------------------------------

func (a *api) listModels(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var (
		models []domain.Model
		err    error
	)
	if r.URL.Query().Get("all") == "true" {
		models, err = a.providers.ListAllModels(r.Context(), actor)
	} else {
		models, err = a.providers.ListModels(r.Context(), actor)
	}
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]modelDTO, 0, len(models))
	for _, model := range models {
		items = append(items, toModelDTO(model))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

// ---- provider probe handlers ----------------------------------------------

// validateProviderCredential proves a secret is accepted by the provider
// without persisting it. An upstream rejection is reported as valid=false
// with 200 so a caller can render the result, not treat it as a gateway fault.
func (a *api) validateProviderCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req providerProbeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	err := a.providers.ValidateCredential(r.Context(), actor, req.ProviderKind, req.APIKey)
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": true})
		return
	}
	if errors.Is(err, provider.ErrInvalidProviderCredential) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}
	writeServiceError(w, r, err)
}

// checkProviderHealth runs a live connectivity probe and records the result.
func (a *api) checkProviderHealth(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req providerProbeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	state, err := a.providers.CheckHealth(r.Context(), actor, req.ProviderKind, req.APIKey)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProviderHealthDTO(state))
}

// listProviderHealth returns the last observed state for every provider kind.
func (a *api) listProviderHealth(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	states, err := a.providers.ListHealth(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]providerHealthDTO, 0, len(states))
	for _, s := range states {
		items = append(items, toProviderHealthDTO(s))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

// syncProviderModels refreshes the global model catalog from the live provider.
func (a *api) syncProviderModels(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req providerProbeRequest
	_ = decodeJSON(w, r, &req)

	var allModels []domain.Model
	if req.ProviderKind != "" {
		if req.APIKey != "" {
			models, err := a.providers.SyncModels(r.Context(), actor, req.ProviderKind, req.APIKey)
			if err != nil {
				writeServiceError(w, r, err)
				return
			}
			allModels = append(allModels, models...)
		} else {
			// Find connected active account for this provider
			accounts, err := a.providers.ListAccounts(r.Context(), actor)
			if err != nil {
				writeServiceError(w, r, err)
				return
			}
			var syncedForKind bool
			for _, acc := range accounts {
				if acc.ProviderKind == req.ProviderKind && acc.Status == domain.ProviderStatusActive {
					models, err := a.providers.SyncAccountModels(r.Context(), actor, acc.ID)
					if err != nil {
						writeServiceError(w, r, err)
						return
					}
					allModels = append(allModels, models...)
					syncedForKind = true
					break
				}
			}
			if !syncedForKind {
				writeServiceError(w, r, fmt.Errorf("%w: api_key or connected active provider account required for %s", repositories.ErrInvalid, req.ProviderKind))
				return
			}
		}
	} else {
		// Sync across all connected active provider accounts for this organization
		accounts, err := a.providers.ListAccounts(r.Context(), actor)
		if err == nil {
			for _, acc := range accounts {
				if acc.Status == domain.ProviderStatusActive {
					models, err := a.providers.SyncAccountModels(r.Context(), actor, acc.ID)
					if err == nil {
						allModels = append(allModels, models...)
					}
				}
			}
		}
	}

	items := make([]modelDTO, 0, len(allModels))
	for _, m := range allModels {
		items = append(items, toModelDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}
