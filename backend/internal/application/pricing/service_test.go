package pricing_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/pricing"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

var (
	testOrgID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	testActorAdmin = domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         uuid.MustParse("00000000-0000-0000-0000-0000000000a1"),
		OrganizationID: testOrgID,
		Role:           domain.RoleOwner,
		DisplayName:    "Admin",
	}
	testActorMember = domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         uuid.MustParse("00000000-0000-0000-0000-0000000000b1"),
		OrganizationID: testOrgID,
		Role:           domain.RoleMember,
		DisplayName:    "Member",
	}
)

type memoryPricingModelRepo struct {
	mu     sync.RWMutex
	models map[uuid.UUID]domain.PricingModel
}

func newMemoryPricingModelRepo() *memoryPricingModelRepo {
	return &memoryPricingModelRepo{
		models: make(map[uuid.UUID]domain.PricingModel),
	}
}

func (r *memoryPricingModelRepo) Upsert(ctx context.Context, orgID uuid.UUID, input domain.NewPricingModel) (domain.PricingModel, error) {
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

func (r *memoryPricingModelRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.PricingModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.models[id]
	if !ok || m.OrganizationID != orgID {
		return domain.PricingModel{}, repositories.ErrNotFound
	}
	return m, nil
}

func (r *memoryPricingModelRepo) GetBySlug(ctx context.Context, orgID uuid.UUID, modelSlug string) (domain.PricingModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.models {
		if m.OrganizationID == orgID && m.ModelSlug == modelSlug {
			return m, nil
		}
	}
	return domain.PricingModel{}, repositories.ErrNotFound
}

func (r *memoryPricingModelRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.PricingModel, error) {
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

type memoryPricingVersionRepo struct {
	mu       sync.RWMutex
	versions []domain.PricingVersion
}

func newMemoryPricingVersionRepo() *memoryPricingVersionRepo {
	return &memoryPricingVersionRepo{
		versions: make([]domain.PricingVersion, 0),
	}
}

func (r *memoryPricingVersionRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewPricingVersion) (domain.PricingVersion, error) {
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

func (r *memoryPricingVersionRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.versions {
		if v.OrganizationID == orgID && v.ID == id {
			return v, nil
		}
	}
	return domain.PricingVersion{}, repositories.ErrNotFound
}

func (r *memoryPricingVersionRepo) GetByVersion(ctx context.Context, orgID, pricingModelID uuid.UUID, version int32) (domain.PricingVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.versions {
		if v.OrganizationID == orgID && v.PricingModelID == pricingModelID && v.Version == version {
			return v, nil
		}
	}
	return domain.PricingVersion{}, repositories.ErrNotFound
}

func (r *memoryPricingVersionRepo) GetLatest(ctx context.Context, orgID, pricingModelID uuid.UUID) (domain.PricingVersion, error) {
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

func (r *memoryPricingVersionRepo) GetEffective(ctx context.Context, orgID, pricingModelID uuid.UUID, at time.Time) (domain.PricingVersion, error) {
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

func (r *memoryPricingVersionRepo) List(ctx context.Context, orgID, pricingModelID uuid.UUID) ([]domain.PricingVersion, error) {
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

type memoryAuditRepo struct {
	mu      sync.Mutex
	entries []domain.NewAuditLog
}

func (a *memoryAuditRepo) Append(ctx context.Context, orgID uuid.UUID, entry domain.NewAuditLog) (domain.AuditLog, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return domain.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Action:         entry.Action,
		ResourceType:   entry.ResourceType,
		ResourceID:     entry.ResourceID,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

func (a *memoryAuditRepo) List(ctx context.Context, orgID uuid.UUID, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, error) {
	return nil, nil
}

func (a *memoryAuditRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.AuditLog, error) {
	return domain.AuditLog{}, nil
}

func (a *memoryAuditRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	return 0, nil
}

func setupTestService() (*pricing.Service, *memoryPricingModelRepo, *memoryPricingVersionRepo, *memoryAuditRepo) {
	models := newMemoryPricingModelRepo()
	versions := newMemoryPricingVersionRepo()
	audits := &memoryAuditRepo{}
	svc := pricing.NewService(pricing.ServiceDeps{
		PricingModels:   models,
		PricingVersions: versions,
		Audits:          audits,
	})
	return svc, models, versions, audits
}

func TestPricingService_CreateAndRetrieve(t *testing.T) {
	svc, _, _, audits := setupTestService()
	ctx := context.Background()

	// 1. Create model
	model, err := svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "gpt-4o",
		ProviderKind: "openai",
		DisplayName:  "GPT-4 Omni",
		Currency:     "USD",
	})
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	if model.ModelSlug != "gpt-4o" || model.Currency != "USD" {
		t.Errorf("unexpected model: %+v", model)
	}

	// 2. Create version
	effFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1, err := svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:           5000,
		OutputUnitPriceMicros:          15000,
		CacheReadInputTokenPriceMicros: 2500,
		EffectiveFrom:                  effFrom,
	})
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}
	if v1.Version != 1 || v1.InputUnitPriceMicros != 5000 || v1.CacheReadInputTokenPriceMicros != 2500 {
		t.Errorf("unexpected version: %+v", v1)
	}

	// 3. List models
	list, err := svc.ListModels(ctx, testActorAdmin)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if len(list) != 1 || list[0].LatestVersion == nil || list[0].LatestVersion.Version != 1 {
		t.Fatalf("expected 1 model with latest version, got %+v", list)
	}

	// 4. Retrieve model and versions
	retrievedModel, versions, err := svc.GetModel(ctx, testActorAdmin, "gpt-4o")
	if err != nil {
		t.Fatalf("GetModel failed: %v", err)
	}
	if retrievedModel.ID != model.ID || len(versions) != 1 {
		t.Errorf("GetModel mismatch: %+v, versions=%d", retrievedModel, len(versions))
	}

	// 5. Verify audit logs
	if len(audits.entries) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(audits.entries))
	} else {
		if audits.entries[0].Action != "pricing.model_created" {
			t.Errorf("expected pricing.model_created, got %s", audits.entries[0].Action)
		}
		if audits.entries[1].Action != "pricing.version_created" {
			t.Errorf("expected pricing.version_created, got %s", audits.entries[1].Action)
		}
	}
}

func TestPricingService_Validation(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Invalid provider kind
	_, err := svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "test-model",
		ProviderKind: "unsupported-provider",
		Currency:     "USD",
	})
	if !errors.Is(err, domain.ErrInvalidProviderKind) {
		t.Errorf("expected ErrInvalidProviderKind, got %v", err)
	}

	// Invalid currency
	_, err = svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "test-model",
		ProviderKind: "openai",
		Currency:     "INVALID_CURRENCY",
	})
	if !errors.Is(err, domain.ErrInvalidCurrency) {
		t.Errorf("expected ErrInvalidCurrency, got %v", err)
	}

	// Create valid model for version tests
	model, err := svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "gpt-3.5-turbo",
		ProviderKind: "openai",
		Currency:     "USD",
	})
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}

	// Negative price
	_, err = svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:  -10,
		OutputUnitPriceMicros: 100,
	})
	if !errors.Is(err, domain.ErrNegativePrice) {
		t.Errorf("expected ErrNegativePrice, got %v", err)
	}

	// Unsupported token category (prompt caching on non-supporting model gpt-3.5-turbo)
	_, err = svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:           1500,
		OutputUnitPriceMicros:          2000,
		CacheReadInputTokenPriceMicros: 500,
	})
	if !errors.Is(err, domain.ErrUnsupportedTokenCategory) {
		t.Errorf("expected ErrUnsupportedTokenCategory for gpt-3.5-turbo, got %v", err)
	}

	// Invalid effective window (effective_to <= effective_from)
	tFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	tTo := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:  1500,
		OutputUnitPriceMicros: 2000,
		EffectiveFrom:         tFrom,
		EffectiveTo:           &tTo,
	})
	if !errors.Is(err, domain.ErrInvalidEffectiveWindow) {
		t.Errorf("expected ErrInvalidEffectiveWindow, got %v", err)
	}
}

func TestPricingService_VersionHistoryAndEffectiveTime(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	model, err := svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "claude-3-5-sonnet-20241022",
		ProviderKind: "anthropic",
		Currency:     "USD",
	})
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}

	// Version 1: effective Jan 2026 to June 2026
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	v1, err := svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:           3000,
		OutputUnitPriceMicros:          15000,
		CacheReadInputTokenPriceMicros: 300,
		EffectiveFrom:                  t1,
		EffectiveTo:                    &t2,
	})
	if err != nil {
		t.Fatalf("CreateVersion v1 failed: %v", err)
	}
	if v1.Version != 1 {
		t.Errorf("expected version 1, got %d", v1.Version)
	}

	// Version 2: effective June 2026 onwards
	v2, err := svc.CreateVersion(ctx, testActorAdmin, model.ID.String(), pricing.CreateVersionInput{
		InputUnitPriceMicros:           2500,
		OutputUnitPriceMicros:          12500,
		CacheReadInputTokenPriceMicros: 250,
		EffectiveFrom:                  t2,
	})
	if err != nil {
		t.Fatalf("CreateVersion v2 failed: %v", err)
	}
	if v2.Version != 2 {
		t.Errorf("expected version 2, got %d", v2.Version)
	}

	// Retrieve specific version 1
	gotV1, err := svc.GetVersion(ctx, testActorAdmin, model.ModelSlug, 1)
	if err != nil {
		t.Fatalf("GetVersion 1 failed: %v", err)
	}
	if gotV1.InputUnitPriceMicros != 3000 {
		t.Errorf("expected v1 input 3000, got %d", gotV1.InputUnitPriceMicros)
	}

	// Cost calculation in March 2026 (should resolve Version 1)
	march := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	costMarch, err := svc.CalculateCost(ctx, testActorAdmin, pricing.CalculateCostRequest{
		ModelSlug:         model.ModelSlug,
		PromptTokens:      1000,
		CompletionTokens:  500,
		CachedInputTokens: 200,
		EffectiveAt:       &march,
	})
	if err != nil {
		t.Fatalf("CalculateCost march failed: %v", err)
	}
	if costMarch.PricingVersion.Version != 1 {
		t.Errorf("expected March to use version 1, got %d", costMarch.PricingVersion.Version)
	}
	// Expected cost: non-cached input (800 * 3000 = 2,400,000) + cached input (200 * 300 = 60,000) + output (500 * 15000 = 7,500,000) = 9,960,000 micros
	var expectedCostMarch int64 = (800 * 3000) + (200 * 300) + (500 * 15000)
	if costMarch.CostMicros != expectedCostMarch {
		t.Errorf("cost = %d, want %d", costMarch.CostMicros, expectedCostMarch)
	}

	// Cost calculation in July 2026 (should resolve Version 2)
	july := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	costJuly, err := svc.CalculateCost(ctx, testActorAdmin, pricing.CalculateCostRequest{
		ModelSlug:         model.ModelSlug,
		PromptTokens:      1000,
		CompletionTokens:  500,
		CachedInputTokens: 200,
		EffectiveAt:       &july,
	})
	if err != nil {
		t.Fatalf("CalculateCost july failed: %v", err)
	}
	if costJuly.PricingVersion.Version != 2 {
		t.Errorf("expected July to use version 2, got %d", costJuly.PricingVersion.Version)
	}
	// Expected cost: non-cached input (800 * 2500 = 2,000,000) + cached input (200 * 250 = 50,000) + output (500 * 12500 = 6,250,000) = 8,300,000 micros
	var expectedCostJuly int64 = (800 * 2500) + (200 * 250) + (500 * 12500)
	if costJuly.CostMicros != expectedCostJuly {
		t.Errorf("cost = %d, want %d", costJuly.CostMicros, expectedCostJuly)
	}

	// Historical cost calculation must remain reproducible when targeting version 1 explicitly
	v1Explicit := int32(1)
	costV1Explicit, err := svc.CalculateCost(ctx, testActorAdmin, pricing.CalculateCostRequest{
		ModelSlug:         model.ModelSlug,
		PromptTokens:      1000,
		CompletionTokens:  500,
		CachedInputTokens: 200,
		Version:           &v1Explicit,
		EffectiveAt:       &july, // even if effectiveAt is July, version=1 takes precedence
	})
	if err != nil {
		t.Fatalf("CalculateCost explicit version 1 failed: %v", err)
	}
	if costV1Explicit.CostMicros != expectedCostMarch || costV1Explicit.PricingVersion.Version != 1 {
		t.Errorf("historical version reproducibility violated: got %d (version %d), want %d",
			costV1Explicit.CostMicros, costV1Explicit.PricingVersion.Version, expectedCostMarch)
	}
}

func TestPricingService_UnknownPricingAndMissingTokens(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Unknown model must explicitly report ErrPricingUnavailable, not zero cost
	_, err := svc.CalculateCost(ctx, testActorAdmin, pricing.CalculateCostRequest{
		ModelSlug:        "unknown-unpriced-model",
		PromptTokens:     100,
		CompletionTokens: 50,
	})
	if !errors.Is(err, domain.ErrPricingUnavailable) {
		t.Fatalf("expected ErrPricingUnavailable for unpriced model, got: %v", err)
	}

	// Negative tokens must return ErrInvalidTokenCount
	_, err = svc.CreateModel(ctx, testActorAdmin, domain.NewPricingModel{
		ModelSlug:    "gemini-1.5-flash",
		ProviderKind: "gemini",
		Currency:     "USD",
	})
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	_, err = svc.CreateVersion(ctx, testActorAdmin, "gemini-1.5-flash", pricing.CreateVersionInput{
		InputUnitPriceMicros:  75,
		OutputUnitPriceMicros: 300,
	})
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	_, err = svc.CalculateCost(ctx, testActorAdmin, pricing.CalculateCostRequest{
		ModelSlug:        "gemini-1.5-flash",
		PromptTokens:     -5,
		CompletionTokens: 10,
	})
	if !errors.Is(err, domain.ErrInvalidTokenCount) {
		t.Errorf("expected ErrInvalidTokenCount for negative prompt tokens, got %v", err)
	}
}

func TestPricingService_Authorization(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Member role has no billing.manage or billing.read permissions
	_, err := svc.CreateModel(ctx, testActorMember, domain.NewPricingModel{
		ModelSlug:    "claude-3-opus",
		ProviderKind: "anthropic",
		Currency:     "USD",
	})
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("expected ErrForbidden for member CreateModel, got %v", err)
	}

	_, err = svc.ListModels(ctx, testActorMember)
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("expected ErrForbidden for member ListModels, got %v", err)
	}

	_, _, err = svc.GetModel(ctx, testActorMember, "claude-3-opus")
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("expected ErrForbidden for member GetModel, got %v", err)
	}

	_, err = svc.CreateVersion(ctx, testActorMember, "claude-3-opus", pricing.CreateVersionInput{
		InputUnitPriceMicros:  1000,
		OutputUnitPriceMicros: 5000,
	})
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("expected ErrForbidden for member CreateVersion, got %v", err)
	}

	_, err = svc.CalculateCost(ctx, testActorMember, pricing.CalculateCostRequest{
		ModelSlug:        "claude-3-opus",
		PromptTokens:     100,
		CompletionTokens: 100,
	})
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Errorf("expected ErrForbidden for member CalculateCost, got %v", err)
	}
}
