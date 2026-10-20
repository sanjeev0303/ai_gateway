package controlapi

import (
	"net/http"
	"strconv"
	"time"

	"ai-governance-saas/backend/internal/application/pricing"
	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/platform/httpx"
)

func registerPricingRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/pricing", a.requireAuth(a.listPricingModels))
	mux.Handle("POST /v1/pricing", a.requireAuth(a.createPricingModel))
	mux.Handle("GET /v1/pricing/{idOrSlug}", a.requireAuth(a.getPricingModel))
	mux.Handle("POST /v1/pricing/{idOrSlug}/versions", a.requireAuth(a.createPricingVersion))
	mux.Handle("GET /v1/pricing/{idOrSlug}/versions/{version}", a.requireAuth(a.getPricingVersion))
	mux.Handle("POST /v1/pricing/calculate-cost", a.requireAuth(a.calculatePricingCost))
}

// pricingVersionDTO is the wire representation of a versioned price sheet.
type pricingVersionDTO struct {
	ID                             string  `json:"id"`
	PricingModelID                 string  `json:"pricing_model_id"`
	Version                        int32   `json:"version"`
	InputUnitPriceMicros           int64   `json:"input_unit_price_micros"`
	OutputUnitPriceMicros          int64   `json:"output_unit_price_micros"`
	CacheReadInputTokenPriceMicros int64   `json:"cache_read_input_token_price_micros"`
	EffectiveFrom                  string  `json:"effective_from"`
	EffectiveTo                    *string `json:"effective_to,omitempty"`
	CreatedAt                      string  `json:"created_at"`
}

// pricingModelDTO is the wire representation of a pricing model.
type pricingModelDTO struct {
	ID            string             `json:"id"`
	ModelSlug     string             `json:"model_slug"`
	ProviderKind  string             `json:"provider_kind"`
	DisplayName   string             `json:"display_name"`
	Currency      string             `json:"currency"`
	Status        string             `json:"status"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
	LatestVersion *pricingVersionDTO `json:"latest_version,omitempty"`
}

type pricingModelDetailDTO struct {
	Model    pricingModelDTO     `json:"model"`
	Versions []pricingVersionDTO `json:"versions"`
}

type createPricingModelRequest struct {
	ModelSlug    string `json:"model_slug"`
	ProviderKind string `json:"provider_kind"`
	DisplayName  string `json:"display_name,omitempty"`
	Currency     string `json:"currency"`
}

type createPricingVersionRequest struct {
	InputUnitPriceMicros           int64      `json:"input_unit_price_micros"`
	OutputUnitPriceMicros          int64      `json:"output_unit_price_micros"`
	CacheReadInputTokenPriceMicros int64      `json:"cache_read_input_token_price_micros,omitempty"`
	EffectiveFrom                  *time.Time `json:"effective_from,omitempty"`
	EffectiveTo                    *time.Time `json:"effective_to,omitempty"`
}

type calculateCostRequest struct {
	ModelSlug         string     `json:"model_slug"`
	ProviderKind      string     `json:"provider_kind,omitempty"`
	PromptTokens      int64      `json:"prompt_tokens"`
	CompletionTokens  int64      `json:"completion_tokens"`
	CachedInputTokens int64      `json:"cached_input_tokens,omitempty"`
	Version           *int32     `json:"version,omitempty"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
}

type calculateCostResponse struct {
	CostMicros        int64             `json:"cost_micros"`
	Currency          string            `json:"currency"`
	PricingVersion    pricingVersionDTO `json:"pricing_version"`
	PromptTokens      int64             `json:"prompt_tokens"`
	CompletionTokens  int64             `json:"completion_tokens"`
	CachedInputTokens int64             `json:"cached_input_tokens"`
}

func toPricingVersionDTO(pv domain.PricingVersion) pricingVersionDTO {
	var effTo *string
	if pv.EffectiveTo != nil {
		v := ts(*pv.EffectiveTo)
		effTo = &v
	}
	return pricingVersionDTO{
		ID:                             pv.ID.String(),
		PricingModelID:                 pv.PricingModelID.String(),
		Version:                        pv.Version,
		InputUnitPriceMicros:           pv.InputUnitPriceMicros,
		OutputUnitPriceMicros:          pv.OutputUnitPriceMicros,
		CacheReadInputTokenPriceMicros: pv.CacheReadInputTokenPriceMicros,
		EffectiveFrom:                  ts(pv.EffectiveFrom),
		EffectiveTo:                    effTo,
		CreatedAt:                      ts(pv.CreatedAt),
	}
}

func toPricingModelDTO(m domain.PricingModel, latest *domain.PricingVersion) pricingModelDTO {
	dto := pricingModelDTO{
		ID:           m.ID.String(),
		ModelSlug:    m.ModelSlug,
		ProviderKind: m.ProviderKind,
		DisplayName:  m.DisplayName,
		Currency:     m.Currency,
		Status:       m.Status,
		CreatedAt:    ts(m.CreatedAt),
		UpdatedAt:    ts(m.UpdatedAt),
	}
	if latest != nil {
		ldto := toPricingVersionDTO(*latest)
		dto.LatestVersion = &ldto
	}
	return dto
}

func (a *api) listPricingModels(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	items, err := a.pricing.ListModels(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	dtos := make([]pricingModelDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toPricingModelDTO(item.Model, item.LatestVersion))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": dtos})
}

func (a *api) getPricingModel(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	idOrSlug := r.PathValue("idOrSlug")
	if idOrSlug == "" {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_argument", "missing pricing model identifier"))
		return
	}

	model, versions, err := a.pricing.GetModel(r.Context(), actor, idOrSlug)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	vdtos := make([]pricingVersionDTO, 0, len(versions))
	var latest *domain.PricingVersion
	for i := range versions {
		if i == 0 {
			latest = &versions[i]
		}
		vdtos = append(vdtos, toPricingVersionDTO(versions[i]))
	}

	detail := pricingModelDetailDTO{
		Model:    toPricingModelDTO(model, latest),
		Versions: vdtos,
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (a *api) createPricingModel(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var body createPricingModelRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, r, err)
		return
	}

	model, err := a.pricing.CreateModel(r.Context(), actor, domain.NewPricingModel{
		ModelSlug:    body.ModelSlug,
		ProviderKind: body.ProviderKind,
		DisplayName:  body.DisplayName,
		Currency:     body.Currency,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toPricingModelDTO(model, nil))
}

func (a *api) createPricingVersion(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	idOrSlug := r.PathValue("idOrSlug")
	if idOrSlug == "" {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_argument", "missing pricing model identifier"))
		return
	}

	var body createPricingVersionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, r, err)
		return
	}

	effFrom := time.Now().UTC()
	if body.EffectiveFrom != nil && !body.EffectiveFrom.IsZero() {
		effFrom = body.EffectiveFrom.UTC()
	}

	pv, err := a.pricing.CreateVersion(r.Context(), actor, idOrSlug, pricing.CreateVersionInput{
		InputUnitPriceMicros:           body.InputUnitPriceMicros,
		OutputUnitPriceMicros:          body.OutputUnitPriceMicros,
		CacheReadInputTokenPriceMicros: body.CacheReadInputTokenPriceMicros,
		EffectiveFrom:                  effFrom,
		EffectiveTo:                    body.EffectiveTo,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toPricingVersionDTO(pv))
}

func (a *api) getPricingVersion(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	idOrSlug := r.PathValue("idOrSlug")
	versionStr := r.PathValue("version")
	if idOrSlug == "" || versionStr == "" {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_argument", "missing model or version identifier"))
		return
	}

	versionNum, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil || versionNum <= 0 {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_argument", "version must be a positive integer"))
		return
	}

	pv, err := a.pricing.GetVersion(r.Context(), actor, idOrSlug, int32(versionNum))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toPricingVersionDTO(pv))
}

func (a *api) calculatePricingCost(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var body calculateCostRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, r, err)
		return
	}

	result, err := a.pricing.CalculateCost(r.Context(), actor, pricing.CalculateCostRequest{
		ModelSlug:         body.ModelSlug,
		ProviderKind:      body.ProviderKind,
		PromptTokens:      body.PromptTokens,
		CompletionTokens:  body.CompletionTokens,
		CachedInputTokens: body.CachedInputTokens,
		Version:           body.Version,
		EffectiveAt:       body.EffectiveAt,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	resp := calculateCostResponse{
		CostMicros:        result.CostMicros,
		Currency:          result.Currency,
		PricingVersion:    toPricingVersionDTO(result.PricingVersion),
		PromptTokens:      result.PromptTokens,
		CompletionTokens:  result.CompletionTokens,
		CachedInputTokens: result.CachedInputTokens,
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
