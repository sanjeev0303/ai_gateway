package controlapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/pricing"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type fakePricingModelRepo struct {
	mu     sync.RWMutex
	models map[uuid.UUID]domain.PricingModel
}

func newFakePricingModelRepo() *fakePricingModelRepo {
	return &fakePricingModelRepo{models: make(map[uuid.UUID]domain.PricingModel)}
}

func (r *fakePricingModelRepo) Upsert(ctx context.Context, orgID uuid.UUID, input domain.NewPricingModel) (domain.PricingModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, m := range r.models {
		if m.OrganizationID == orgID && m.ModelSlug == input.ModelSlug {
			m.DisplayName = input.DisplayName
			m.Currency = input.Currency
			m.ProviderKind = input.ProviderKind
			m.UpdatedAt = time.Now().UTC()
			r.models[id] = m
			return m, nil
		}
	}

	m := domain.PricingModel{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ModelSlug:      input.ModelSlug,
		ProviderKind:   input.ProviderKind,
		DisplayName:    input.DisplayName,
		Currency:       input.Currency,
		Status:         domain.PricingModelStatusActive,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	r.models[m.ID] = m
	return m, nil
}

func (r *fakePricingModelRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.PricingModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.models[id]
	if !ok || m.OrganizationID != orgID {
		return domain.PricingModel{}, repositories.ErrNotFound
	}
	return m, nil
}

func (r *fakePricingModelRepo) GetBySlug(ctx context.Context, orgID uuid.UUID, modelSlug string) (domain.PricingModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.models {
		if m.OrganizationID == orgID && m.ModelSlug == modelSlug {
			return m, nil
		}
	}
	return domain.PricingModel{}, repositories.ErrNotFound
}

func (r *fakePricingModelRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.PricingModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []domain.PricingModel
	for _, m := range r.models {
		if m.OrganizationID == orgID {
			list = append(list, m)
		}
	}
	return list, nil
}

type fakePricingVersionRepo struct {
	mu       sync.RWMutex
	versions []domain.PricingVersion
}

func newFakePricingVersionRepo() *fakePricingVersionRepo {
	return &fakePricingVersionRepo{versions: make([]domain.PricingVersion, 0)}
}

func (r *fakePricingVersionRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewPricingVersion) (domain.PricingVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pv := domain.PricingVersion{
		ID:                             uuid.New(),
		PricingModelID:                 input.PricingModelID,
		OrganizationID:                 orgID,
		Version:                        input.Version,
		InputUnitPriceMicros:           input.InputUnitPriceMicros,
		OutputUnitPriceMicros:          input.OutputUnitPriceMicros,
		CacheReadInputTokenPriceMicros: input.CacheReadInputTokenPriceMicros,
		EffectiveFrom:                  input.EffectiveFrom,
		EffectiveTo:                    input.EffectiveTo,
		CreatedAt:                      time.Now().UTC(),
	}
	r.versions = append(r.versions, pv)
	return pv, nil
}

func (r *fakePricingVersionRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.versions {
		if v.OrganizationID == orgID && v.ID == id {
			return v, nil
		}
	}
	return domain.PricingVersion{}, repositories.ErrNotFound
}

func (r *fakePricingVersionRepo) GetByVersion(ctx context.Context, orgID, pricingModelID uuid.UUID, version int32) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.versions {
		if v.OrganizationID == orgID && v.PricingModelID == pricingModelID && v.Version == version {
			return v, nil
		}
	}
	return domain.PricingVersion{}, repositories.ErrNotFound
}

func (r *fakePricingVersionRepo) GetLatest(ctx context.Context, orgID, pricingModelID uuid.UUID) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *domain.PricingVersion
	for i := range r.versions {
		v := &r.versions[i]
		if v.OrganizationID == orgID && v.PricingModelID == pricingModelID {
			if latest == nil || v.Version > latest.Version {
				latest = v
			}
		}
	}
	if latest == nil {
		return domain.PricingVersion{}, repositories.ErrNotFound
	}
	return *latest, nil
}

func (r *fakePricingVersionRepo) GetEffective(ctx context.Context, orgID, pricingModelID uuid.UUID, at time.Time) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *domain.PricingVersion
	for i := range r.versions {
		v := &r.versions[i]
		if v.OrganizationID != orgID || v.PricingModelID != pricingModelID {
			continue
		}
		if v.EffectiveFrom.After(at) {
			continue
		}
		if v.EffectiveTo != nil && !v.EffectiveTo.After(at) {
			continue
		}
		if best == nil || v.EffectiveFrom.After(best.EffectiveFrom) || (v.EffectiveFrom.Equal(best.EffectiveFrom) && v.Version > best.Version) {
			best = v
		}
	}
	if best == nil {
		return domain.PricingVersion{}, repositories.ErrNotFound
	}
	return *best, nil
}

func (r *fakePricingVersionRepo) List(ctx context.Context, orgID, pricingModelID uuid.UUID) ([]domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []domain.PricingVersion
	for _, v := range r.versions {
		if v.OrganizationID == orgID && v.PricingModelID == pricingModelID {
			list = append(list, v)
		}
	}
	return list, nil
}

func newPricingServer(t *testing.T, st *controlStubs, models *fakePricingModelRepo, versions *fakePricingVersionRepo) *httptest.Server {
	t.Helper()
	pricingSvc := pricing.NewService(pricing.ServiceDeps{
		PricingModels:   models,
		PricingVersions: versions,
	})
	return buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		Pricing:         pricingSvc,
		DevLoginEnabled: true,
	})
}

func TestPricingRoutesRequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newPricingServer(t, st, newFakePricingModelRepo(), newFakePricingVersionRepo())

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/pricing"},
		{http.MethodPost, "/v1/pricing"},
		{http.MethodGet, "/v1/pricing/gpt-4o"},
		{http.MethodPost, "/v1/pricing/gpt-4o/versions"},
		{http.MethodGet, "/v1/pricing/gpt-4o/versions/1"},
		{http.MethodPost, "/v1/pricing/calculate-cost"},
	} {
		resp := doJSON(t, srv, tc.method, tc.path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s: got %d, want 401", tc.method, tc.path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestPricingRoutesForbiddenForMember(t *testing.T) {
	st := newControlStubs()
	srv := newPricingServer(t, st, newFakePricingModelRepo(), newFakePricingVersionRepo())

	for _, tc := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/v1/pricing", nil},
		{http.MethodPost, "/v1/pricing", map[string]any{"model_slug": "gpt-4o", "provider_kind": "openai", "currency": "USD"}},
		{http.MethodGet, "/v1/pricing/gpt-4o", nil},
		{http.MethodPost, "/v1/pricing/gpt-4o/versions", map[string]any{"input_unit_price_micros": 1000, "output_unit_price_micros": 2000}},
		{http.MethodGet, "/v1/pricing/gpt-4o/versions/1", nil},
		{http.MethodPost, "/v1/pricing/calculate-cost", map[string]any{"model_slug": "gpt-4o", "prompt_tokens": 100, "completion_tokens": 50}},
	} {
		resp := doJSON(t, srv, tc.method, tc.path, tc.body, bearerHeader(st, domain.RoleMember))
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s %s: got %d, want 403 Forbidden", tc.method, tc.path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestPricingCreateListAndGet(t *testing.T) {
	st := newControlStubs()
	models := newFakePricingModelRepo()
	versions := newFakePricingVersionRepo()
	srv := newPricingServer(t, st, models, versions)

	// 1. Create model
	createModelBody := map[string]any{
		"model_slug":    "gpt-4o",
		"provider_kind": "openai",
		"display_name":  "GPT-4 Omni",
		"currency":      "USD",
	}
	resp := doJSON(t, srv, http.MethodPost, "/v1/pricing", createModelBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /v1/pricing: got %d, want 201", resp.StatusCode)
	}
	var createdModel pricingModelDTO
	decodeBody(t, resp, &createdModel)
	if createdModel.ModelSlug != "gpt-4o" || createdModel.Currency != "USD" {
		t.Fatalf("unexpected created model: %+v", createdModel)
	}

	// 2. Create version with cached tokens
	createVersionBody := map[string]any{
		"input_unit_price_micros":            5000,
		"output_unit_price_micros":           15000,
		"cache_read_input_token_price_micros": 2500,
	}
	resp = doJSON(t, srv, http.MethodPost, "/v1/pricing/gpt-4o/versions", createVersionBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /v1/pricing/gpt-4o/versions: got %d, want 201", resp.StatusCode)
	}
	var createdVersion pricingVersionDTO
	decodeBody(t, resp, &createdVersion)
	if createdVersion.Version != 1 || createdVersion.CacheReadInputTokenPriceMicros != 2500 {
		t.Fatalf("unexpected version: %+v", createdVersion)
	}

	// 3. List pricing
	resp = doJSON(t, srv, http.MethodGet, "/v1/pricing", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/pricing: got %d, want 200", resp.StatusCode)
	}
	var listResp struct {
		Data []pricingModelDTO `json:"data"`
	}
	decodeBody(t, resp, &listResp)
	if len(listResp.Data) != 1 || listResp.Data[0].LatestVersion == nil || listResp.Data[0].LatestVersion.Version != 1 {
		t.Fatalf("unexpected list response: %+v", listResp)
	}

	// 4. Get pricing model detail
	resp = doJSON(t, srv, http.MethodGet, "/v1/pricing/gpt-4o", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/pricing/gpt-4o: got %d, want 200", resp.StatusCode)
	}
	var detailResp pricingModelDetailDTO
	decodeBody(t, resp, &detailResp)
	if detailResp.Model.ModelSlug != "gpt-4o" || len(detailResp.Versions) != 1 {
		t.Fatalf("unexpected detail: %+v", detailResp)
	}

	// 5. Get specific version
	resp = doJSON(t, srv, http.MethodGet, "/v1/pricing/gpt-4o/versions/1", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/pricing/gpt-4o/versions/1: got %d, want 200", resp.StatusCode)
	}
	var v1Resp pricingVersionDTO
	decodeBody(t, resp, &v1Resp)
	if v1Resp.Version != 1 || v1Resp.InputUnitPriceMicros != 5000 {
		t.Fatalf("unexpected v1: %+v", v1Resp)
	}
}

func TestPricingValidationAndCostCalculation(t *testing.T) {
	st := newControlStubs()
	models := newFakePricingModelRepo()
	versions := newFakePricingVersionRepo()
	srv := newPricingServer(t, st, models, versions)

	// Create model
	resp := doJSON(t, srv, http.MethodPost, "/v1/pricing", map[string]any{
		"model_slug":    "claude-3-5-sonnet",
		"provider_kind": "anthropic",
		"currency":      "USD",
	}, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create claude model: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Negative price should return 400
	resp = doJSON(t, srv, http.MethodPost, "/v1/pricing/claude-3-5-sonnet/versions", map[string]any{
		"input_unit_price_micros":  -100,
		"output_unit_price_micros": 500,
	}, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("negative price: got %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// Create valid version
	resp = doJSON(t, srv, http.MethodPost, "/v1/pricing/claude-3-5-sonnet/versions", map[string]any{
		"input_unit_price_micros":            3000,
		"output_unit_price_micros":           15000,
		"cache_read_input_token_price_micros": 300,
	}, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create valid version: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Calculate cost with cached tokens
	calcBody := map[string]any{
		"model_slug":          "claude-3-5-sonnet",
		"prompt_tokens":       1000,
		"completion_tokens":   200,
		"cached_input_tokens": 400,
	}
	resp = doJSON(t, srv, http.MethodPost, "/v1/pricing/calculate-cost", calcBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("calculate cost: got %d, want 200", resp.StatusCode)
	}
	var calcResp calculateCostResponse
	decodeBody(t, resp, &calcResp)
	// (600 non-cached * 3000 = 1,800,000) + (400 cached * 300 = 120,000) + (200 completion * 15000 = 3,000,000) = 4,920,000 micros
	var expectedCost int64 = (600 * 3000) + (400 * 300) + (200 * 15000)
	if calcResp.CostMicros != expectedCost {
		t.Fatalf("cost = %d, want %d", calcResp.CostMicros, expectedCost)
	}

	// Calculate cost for unknown model returns 503 unavailable
	resp = doJSON(t, srv, http.MethodPost, "/v1/pricing/calculate-cost", map[string]any{
		"model_slug":        "unknown-unpriced-model",
		"prompt_tokens":     100,
		"completion_tokens": 100,
	}, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unknown model cost: got %d, want 503 Service Unavailable", resp.StatusCode)
	}
	resp.Body.Close()
}
