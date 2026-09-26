package controlapi

import (
	"net/http"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
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
	ID           string `json:"id"`
	ProviderKind string `json:"provider_kind"`
	ModelSlug    string `json:"model_slug"`
	DisplayName  string `json:"display_name"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
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
	return modelDTO{
		ID:           m.ID.String(),
		ProviderKind: m.ProviderKind,
		ModelSlug:    m.ModelSlug,
		DisplayName:  m.DisplayName,
		IsActive:     m.IsActive,
		CreatedAt:    ts(m.CreatedAt),
		UpdatedAt:    ts(m.UpdatedAt),
	}
}

// ---- request bodies -------------------------------------------------------

type createProviderAccountRequest struct {
	ProviderKind string `json:"provider_kind"`
	Name         string `json:"name"`
	BaseURL      string `json:"base_url,omitempty"`
}

type createProviderCredentialRequest struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Secret string `json:"secret"`
}

type rotateProviderCredentialRequest struct {
	Secret string `json:"secret"`
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
	models, err := a.providers.ListModels(r.Context(), actor)
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
