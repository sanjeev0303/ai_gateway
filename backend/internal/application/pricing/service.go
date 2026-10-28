// Package pricing provides application services for pricing catalog management,
// versioned price sheets, and token cost calculation.
package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// PricingModelWithVersion pairs a pricing model with its latest pricing version.
type PricingModelWithVersion struct {
	Model         domain.PricingModel    `json:"model"`
	LatestVersion *domain.PricingVersion `json:"latest_version,omitempty"`
}

// CreateVersionInput holds parameters for creating a new pricing version.
type CreateVersionInput struct {
	InputUnitPriceMicros           int64      `json:"input_unit_price_micros"`
	OutputUnitPriceMicros          int64      `json:"output_unit_price_micros"`
	CacheReadInputTokenPriceMicros int64      `json:"cache_read_input_token_price_micros"`
	EffectiveFrom                  time.Time  `json:"effective_from"`
	EffectiveTo                    *time.Time `json:"effective_to,omitempty"`
}

// CalculateCostRequest defines parameters to compute request cost.
type CalculateCostRequest struct {
	ModelSlug         string     `json:"model_slug"`
	ProviderKind      string     `json:"provider_kind,omitempty"`
	PromptTokens      int64      `json:"prompt_tokens"`
	CompletionTokens  int64      `json:"completion_tokens"`
	CachedInputTokens int64      `json:"cached_input_tokens,omitempty"`
	Version           *int32     `json:"version,omitempty"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
}

// CalculateCostResult contains calculated cost details.
type CalculateCostResult struct {
	CostMicros            int64                 `json:"cost_micros"`
	InputCostMicros       int64                 `json:"input_cost_micros"`
	OutputCostMicros      int64                 `json:"output_cost_micros"`
	CachedInputCostMicros int64                 `json:"cached_input_cost_micros"`
	SavingsMicros         int64                 `json:"savings_micros"`
	PricingModelFormula   string                `json:"pricing_model_formula,omitempty"`
	Currency              string                `json:"currency"`
	PricingVersion        domain.PricingVersion `json:"pricing_version"`
	PromptTokens          int64                 `json:"prompt_tokens"`
	CompletionTokens      int64                 `json:"completion_tokens"`
	CachedInputTokens     int64                 `json:"cached_input_tokens"`
}

// ServiceDeps contains the repositories needed by the pricing service.
type ServiceDeps struct {
	PricingModels   repositories.PricingModelRepository
	PricingVersions repositories.PricingVersionRepository
	Audits          repositories.AuditLogRepository
}

// Service manages pricing models, versioned price sheets, and cost calculations.
type Service struct {
	pricingModels   repositories.PricingModelRepository
	pricingVersions repositories.PricingVersionRepository
	audits          repositories.AuditLogRepository
}

// NewService constructs a new pricing service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		pricingModels:   deps.PricingModels,
		pricingVersions: deps.PricingVersions,
		audits:          deps.Audits,
	}
}

func (s *Service) checkReadAuth(actor domain.Actor) error {
	if !actor.Can(domain.PermissionBillingRead) && !actor.Can(domain.PermissionProvidersRead) {
		return repositories.ErrForbidden
	}
	return nil
}

func (s *Service) checkWriteAuth(actor domain.Actor) error {
	if !actor.Can(domain.PermissionBillingManage) {
		return repositories.ErrForbidden
	}
	return nil
}

func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceType string, resourceID *uuid.UUID, before, after []byte) {
	if s.audits == nil {
		return
	}
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	_, _ = s.audits.Append(ctx, actor.OrganizationID, entry)
}

// ListModels returns all pricing models for the actor's organization alongside their latest version.
func (s *Service) ListModels(ctx context.Context, actor domain.Actor) ([]PricingModelWithVersion, error) {
	if err := s.checkReadAuth(actor); err != nil {
		return nil, err
	}

	models, err := s.pricingModels.List(ctx, actor.OrganizationID)
	if err != nil {
		return nil, err
	}

	results := make([]PricingModelWithVersion, 0, len(models))
	for _, m := range models {
		item := PricingModelWithVersion{Model: m}
		latest, err := s.pricingVersions.GetLatest(ctx, actor.OrganizationID, m.ID)
		if err == nil {
			item.LatestVersion = &latest
		} else if !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
		results = append(results, item)
	}

	if len(results) == 0 {
		canonicalSpecs := []domain.ProviderPricingSpec{
			{ProviderKind: "openai", ModelSlug: "gpt-4o", DisplayName: "GPT-4o", InputRatePerM: 2.50, OutputRatePerM: 10.00, CachedReadRatePerM: 1.25, SupportsCaching: true},
			{ProviderKind: "openai", ModelSlug: "gpt-4o-mini", DisplayName: "GPT-4o Mini", InputRatePerM: 0.15, OutputRatePerM: 0.60, CachedReadRatePerM: 0.075, SupportsCaching: true},
			{ProviderKind: "openai", ModelSlug: "o1", DisplayName: "o1", InputRatePerM: 15.00, OutputRatePerM: 60.00, CachedReadRatePerM: 7.50, SupportsCaching: true},
			{ProviderKind: "anthropic", ModelSlug: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet", InputRatePerM: 3.00, OutputRatePerM: 15.00, CachedReadRatePerM: 0.30, SupportsCaching: true},
			{ProviderKind: "anthropic", ModelSlug: "claude-3-5-haiku", DisplayName: "Claude 3.5 Haiku", InputRatePerM: 0.80, OutputRatePerM: 4.00, CachedReadRatePerM: 0.08, SupportsCaching: true},
			{ProviderKind: "anthropic", ModelSlug: "claude-3-opus", DisplayName: "Claude 3 Opus", InputRatePerM: 15.00, OutputRatePerM: 75.00, CachedReadRatePerM: 1.50, SupportsCaching: true},
			{ProviderKind: "gemini", ModelSlug: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash", InputRatePerM: 0.075, OutputRatePerM: 0.30, CachedReadRatePerM: 0.01875, SupportsCaching: true},
			{ProviderKind: "gemini", ModelSlug: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro", InputRatePerM: 1.25, OutputRatePerM: 5.00, CachedReadRatePerM: 0.3125, SupportsCaching: true},
			{ProviderKind: "gemini", ModelSlug: "gemini-1.5-flash", DisplayName: "Gemini 1.5 Flash", InputRatePerM: 0.075, OutputRatePerM: 0.30, CachedReadRatePerM: 0.01875, SupportsCaching: true},
			{ProviderKind: "gemini", ModelSlug: "gemini-1.5-pro", DisplayName: "Gemini 1.5 Pro", InputRatePerM: 1.25, OutputRatePerM: 5.00, CachedReadRatePerM: 0.3125, SupportsCaching: true},
			{ProviderKind: "huggingface", ModelSlug: "meta-llama/Llama-3.3-70B-Instruct", DisplayName: "Llama 3.3 70B Instruct", InputRatePerM: 0.40, OutputRatePerM: 0.40, CachedReadRatePerM: 0.40, SupportsCaching: false},
			{ProviderKind: "huggingface", ModelSlug: "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B", DisplayName: "DeepSeek R1 Distill", InputRatePerM: 0.15, OutputRatePerM: 0.60, CachedReadRatePerM: 0.15, SupportsCaching: false},
			{ProviderKind: "huggingface", ModelSlug: "mistralai/Mistral-7B-Instruct-v0.3", DisplayName: "Mistral 7B Instruct", InputRatePerM: 0.05, OutputRatePerM: 0.05, CachedReadRatePerM: 0.05, SupportsCaching: false},
			{ProviderKind: "huggingface", ModelSlug: "Qwen/Qwen2.5-72B-Instruct", DisplayName: "Qwen 2.5 72B", InputRatePerM: 0.40, OutputRatePerM: 0.40, CachedReadRatePerM: 0.40, SupportsCaching: false},
		}
		now := time.Now().UTC()
		for _, sp := range canonicalSpecs {
			mid := uuid.NewSHA1(uuid.NameSpaceOID, []byte(sp.ModelSlug))
			vid := uuid.NewSHA1(uuid.NameSpaceOID, []byte(sp.ModelSlug+"-v1"))
			v := domain.PricingVersion{
				ID:                             vid,
				PricingModelID:                 mid,
				OrganizationID:                 actor.OrganizationID,
				Version:                        1,
				InputUnitPriceMicros:           int64(math.Round(sp.InputRatePerM)),
				OutputUnitPriceMicros:          int64(math.Round(sp.OutputRatePerM)),
				CacheReadInputTokenPriceMicros: int64(math.Round(sp.CachedReadRatePerM)),
				EffectiveFrom:                  now.Add(-30 * 24 * time.Hour),
				CreatedAt:                      now.Add(-30 * 24 * time.Hour),
			}
			results = append(results, PricingModelWithVersion{
				Model: domain.PricingModel{
					ID:             mid,
					OrganizationID: actor.OrganizationID,
					ModelSlug:      sp.ModelSlug,
					ProviderKind:   sp.ProviderKind,
					DisplayName:    sp.DisplayName,
					Currency:       "usd",
					Status:         "active",
					CreatedAt:      now.Add(-30 * 24 * time.Hour),
					UpdatedAt:      now.Add(-30 * 24 * time.Hour),
				},
				LatestVersion: &v,
			})
		}
	}

	return results, nil
}

// GetModel retrieves a single pricing model by ID or slug, along with its complete version history.
func (s *Service) GetModel(ctx context.Context, actor domain.Actor, idOrSlug string) (domain.PricingModel, []domain.PricingVersion, error) {
	if err := s.checkReadAuth(actor); err != nil {
		return domain.PricingModel{}, nil, err
	}

	model, err := s.resolveModel(ctx, actor.OrganizationID, idOrSlug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			if spec, ok := domain.LookupCanonicalPricing("", idOrSlug); ok {
				now := time.Now().UTC()
				mid := uuid.NewSHA1(uuid.NameSpaceOID, []byte(spec.ModelSlug))
				vid := uuid.NewSHA1(uuid.NameSpaceOID, []byte(spec.ModelSlug+"-v1"))
				v := domain.PricingVersion{
					ID:                             vid,
					PricingModelID:                 mid,
					OrganizationID:                 actor.OrganizationID,
					Version:                        1,
					InputUnitPriceMicros:           int64(math.Round(spec.InputRatePerM)),
					OutputUnitPriceMicros:          int64(math.Round(spec.OutputRatePerM)),
					CacheReadInputTokenPriceMicros: int64(math.Round(spec.CachedReadRatePerM)),
					EffectiveFrom:                  now.Add(-30 * 24 * time.Hour),
					CreatedAt:                      now.Add(-30 * 24 * time.Hour),
				}
				return domain.PricingModel{
					ID:             mid,
					OrganizationID: actor.OrganizationID,
					ModelSlug:      spec.ModelSlug,
					ProviderKind:   spec.ProviderKind,
					DisplayName:    spec.DisplayName,
					Currency:       "usd",
					Status:         "active",
					CreatedAt:      now.Add(-30 * 24 * time.Hour),
					UpdatedAt:      now.Add(-30 * 24 * time.Hour),
				}, []domain.PricingVersion{v}, nil
			}
		}
		return domain.PricingModel{}, nil, err
	}

	versions, err := s.pricingVersions.List(ctx, actor.OrganizationID, model.ID)
	if err != nil {
		return domain.PricingModel{}, nil, err
	}

	return model, versions, nil
}

// CreateModel creates or updates a pricing model and records an audit event.
func (s *Service) CreateModel(ctx context.Context, actor domain.Actor, input domain.NewPricingModel) (domain.PricingModel, error) {
	if err := s.checkWriteAuth(actor); err != nil {
		return domain.PricingModel{}, err
	}

	slug := strings.TrimSpace(input.ModelSlug)
	if slug == "" {
		return domain.PricingModel{}, repositories.ErrInvalid
	}
	providerKind := strings.ToLower(strings.TrimSpace(input.ProviderKind))
	if !domain.ValidProviderKind(providerKind) {
		return domain.PricingModel{}, domain.ErrInvalidProviderKind
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if len(currency) != 3 {
		return domain.PricingModel{}, domain.ErrInvalidCurrency
	}

	input.ModelSlug = slug
	input.ProviderKind = providerKind
	input.Currency = currency
	if strings.TrimSpace(input.DisplayName) == "" {
		input.DisplayName = slug
	}

	model, err := s.pricingModels.Upsert(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.PricingModel{}, err
	}

	meta, _ := json.Marshal(map[string]any{
		"model_slug":    model.ModelSlug,
		"provider_kind": model.ProviderKind,
		"currency":      model.Currency,
		"display_name":  model.DisplayName,
	})
	s.audit(ctx, actor, "pricing.model_created", "pricing_model", &model.ID, nil, meta)

	return model, nil
}

// CreateVersion appends a new immutable pricing version to a model, preserving historical versions.
func (s *Service) CreateVersion(ctx context.Context, actor domain.Actor, idOrSlug string, input CreateVersionInput) (domain.PricingVersion, error) {
	if err := s.checkWriteAuth(actor); err != nil {
		return domain.PricingVersion{}, err
	}

	model, err := s.resolveModel(ctx, actor.OrganizationID, idOrSlug)
	if err != nil {
		return domain.PricingVersion{}, err
	}

	if input.InputUnitPriceMicros < 0 || input.OutputUnitPriceMicros < 0 || input.CacheReadInputTokenPriceMicros < 0 {
		return domain.PricingVersion{}, domain.ErrNegativePrice
	}

	if input.CacheReadInputTokenPriceMicros > 0 && !domain.ModelSupportsCachedTokens(model.ProviderKind, model.ModelSlug) {
		return domain.PricingVersion{}, domain.ErrUnsupportedTokenCategory
	}

	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = time.Now().UTC()
	} else {
		input.EffectiveFrom = input.EffectiveFrom.UTC()
	}

	if input.EffectiveTo != nil {
		utc := input.EffectiveTo.UTC()
		input.EffectiveTo = &utc
		if !input.EffectiveTo.After(input.EffectiveFrom) {
			return domain.PricingVersion{}, domain.ErrInvalidEffectiveWindow
		}
	}

	latest, err := s.pricingVersions.GetLatest(ctx, actor.OrganizationID, model.ID)
	var nextVersion int32 = 1
	if err == nil {
		nextVersion = latest.Version + 1
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return domain.PricingVersion{}, err
	}

	pv, err := s.pricingVersions.Create(ctx, actor.OrganizationID, domain.NewPricingVersion{
		PricingModelID:                 model.ID,
		Version:                        nextVersion,
		InputUnitPriceMicros:           input.InputUnitPriceMicros,
		OutputUnitPriceMicros:          input.OutputUnitPriceMicros,
		CacheReadInputTokenPriceMicros: input.CacheReadInputTokenPriceMicros,
		EffectiveFrom:                  input.EffectiveFrom,
		EffectiveTo:                    input.EffectiveTo,
	})
	if err != nil {
		return domain.PricingVersion{}, err
	}

	meta, _ := json.Marshal(map[string]any{
		"pricing_model_id":  model.ID,
		"model_slug":        model.ModelSlug,
		"version":           pv.Version,
		"input_micros":      pv.InputUnitPriceMicros,
		"output_micros":     pv.OutputUnitPriceMicros,
		"cache_read_micros": pv.CacheReadInputTokenPriceMicros,
		"effective_from":    pv.EffectiveFrom,
		"effective_to":      pv.EffectiveTo,
	})
	s.audit(ctx, actor, "pricing.version_created", "pricing_version", &pv.ID, nil, meta)

	return pv, nil
}

// GetVersion retrieves a specific pricing version by model identifier and version number.
func (s *Service) GetVersion(ctx context.Context, actor domain.Actor, idOrSlug string, version int32) (domain.PricingVersion, error) {
	if err := s.checkReadAuth(actor); err != nil {
		return domain.PricingVersion{}, err
	}

	model, err := s.resolveModel(ctx, actor.OrganizationID, idOrSlug)
	if err != nil {
		return domain.PricingVersion{}, err
	}

	pv, err := s.pricingVersions.GetByVersion(ctx, actor.OrganizationID, model.ID, version)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.PricingVersion{}, domain.ErrPricingNotFound
		}
		return domain.PricingVersion{}, err
	}

	return pv, nil
}

// CalculateCost computes total token cost for a model using effective or specified pricing.
func (s *Service) CalculateCost(ctx context.Context, actor domain.Actor, req CalculateCostRequest) (CalculateCostResult, error) {
	if err := s.checkReadAuth(actor); err != nil {
		return CalculateCostResult{}, err
	}

	if req.PromptTokens < 0 || req.CompletionTokens < 0 || req.CachedInputTokens < 0 {
		return CalculateCostResult{}, domain.ErrInvalidTokenCount
	}

	slug := strings.TrimSpace(req.ModelSlug)
	if slug == "" {
		return CalculateCostResult{}, repositories.ErrInvalid
	}

	model, err := s.pricingModels.GetBySlug(ctx, actor.OrganizationID, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			if spec, ok := domain.LookupCanonicalPricing("", slug); ok {
				if req.CachedInputTokens > 0 && !spec.SupportsCaching {
					return CalculateCostResult{}, domain.ErrUnsupportedTokenCategory
				}
				inputRate := spec.InputRatePerM
				outputRate := spec.OutputRatePerM
				formula := fmt.Sprintf("%s standard token pricing: input=$%.2f/1M, output=$%.2f/1M", strings.ToUpper(spec.ProviderKind), inputRate, outputRate)
				if spec.DeepContextThreshold > 0 && req.PromptTokens > spec.DeepContextThreshold {
					if spec.DeepContextInputRatePerM > 0 {
						inputRate = spec.DeepContextInputRatePerM
					}
					if spec.DeepContextOutputRatePerM > 0 {
						outputRate = spec.DeepContextOutputRatePerM
					}
					formula += fmt.Sprintf(" (deep context >128k tier active: input=$%.2f/1M, output=$%.2f/1M)", inputRate, outputRate)
				}
				nonCached := req.PromptTokens - req.CachedInputTokens
				if nonCached < 0 {
					nonCached = 0
				}
				inputCostMicros := int64(math.Round(float64(nonCached) * inputRate))
				outputCostMicros := int64(math.Round(float64(req.CompletionTokens) * outputRate))
				var cachedCostMicros, savingsMicros int64
				if spec.SupportsCaching && req.CachedInputTokens > 0 {
					cachedCostMicros = int64(math.Round(float64(req.CachedInputTokens) * spec.CachedReadRatePerM))
					savingsMicros = int64(math.Round(float64(req.CachedInputTokens) * inputRate)) - cachedCostMicros
					if savingsMicros < 0 {
						savingsMicros = 0
					}
				}
				costMicros := domain.CalculateProviderTokenCost(spec, req.PromptTokens, req.CompletionTokens, req.CachedInputTokens)
				return CalculateCostResult{
					CostMicros:            costMicros,
					InputCostMicros:       inputCostMicros,
					OutputCostMicros:      outputCostMicros,
					CachedInputCostMicros: cachedCostMicros,
					SavingsMicros:         savingsMicros,
					PricingModelFormula:   formula,
					Currency:              "usd",
					PricingVersion: domain.PricingVersion{
						Version:                         1,
						InputUnitPriceMicros:            int64(math.Round(spec.InputRatePerM)),
						OutputUnitPriceMicros:           int64(math.Round(spec.OutputRatePerM)),
						CacheReadInputTokenPriceMicros: int64(math.Round(spec.CachedReadRatePerM)),
					},
					PromptTokens:      req.PromptTokens,
					CompletionTokens:  req.CompletionTokens,
					CachedInputTokens: req.CachedInputTokens,
				}, nil
			}
			return CalculateCostResult{}, domain.ErrPricingUnavailable
		}
		return CalculateCostResult{}, err
	}

	if req.CachedInputTokens > 0 && !domain.ModelSupportsCachedTokens(model.ProviderKind, model.ModelSlug) {
		return CalculateCostResult{}, domain.ErrUnsupportedTokenCategory
	}

	var pv domain.PricingVersion
	if req.Version != nil && *req.Version > 0 {
		pv, err = s.pricingVersions.GetByVersion(ctx, actor.OrganizationID, model.ID, *req.Version)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return CalculateCostResult{}, domain.ErrPricingUnavailable
			}
			return CalculateCostResult{}, err
		}
	} else {
		at := time.Now().UTC()
		if req.EffectiveAt != nil && !req.EffectiveAt.IsZero() {
			at = req.EffectiveAt.UTC()
		}
		pv, err = s.pricingVersions.GetEffective(ctx, actor.OrganizationID, model.ID, at)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return CalculateCostResult{}, domain.ErrPricingUnavailable
			}
			return CalculateCostResult{}, err
		}
	}

	nonCached := req.PromptTokens - req.CachedInputTokens
	if nonCached < 0 {
		nonCached = 0
	}
	var inputCostMicros, cachedCostMicros, savingsMicros int64
	if pv.CacheReadInputTokenPriceMicros > 0 && req.CachedInputTokens > 0 {
		inputCostMicros = nonCached * pv.InputUnitPriceMicros
		cachedCostMicros = req.CachedInputTokens * pv.CacheReadInputTokenPriceMicros
		savingsMicros = (req.CachedInputTokens * pv.InputUnitPriceMicros) - cachedCostMicros
		if savingsMicros < 0 {
			savingsMicros = 0
		}
	} else {
		inputCostMicros = req.PromptTokens * pv.InputUnitPriceMicros
	}
	outputCostMicros := req.CompletionTokens * pv.OutputUnitPriceMicros

	costMicros, err := domain.CalculateCost(pv, req.PromptTokens, req.CompletionTokens, req.CachedInputTokens)
	if err != nil {
		return CalculateCostResult{}, err
	}

	return CalculateCostResult{
		CostMicros:            costMicros,
		InputCostMicros:       inputCostMicros,
		OutputCostMicros:      outputCostMicros,
		CachedInputCostMicros: cachedCostMicros,
		SavingsMicros:         savingsMicros,
		PricingModelFormula:   fmt.Sprintf("Organization customized rate sheet v%d: input=%dµ$/tok, output=%dµ$/tok", pv.Version, pv.InputUnitPriceMicros, pv.OutputUnitPriceMicros),
		Currency:              model.Currency,
		PricingVersion:        pv,
		PromptTokens:          req.PromptTokens,
		CompletionTokens:      req.CompletionTokens,
		CachedInputTokens:     req.CachedInputTokens,
	}, nil
}

func (s *Service) resolveModel(ctx context.Context, orgID uuid.UUID, idOrSlug string) (domain.PricingModel, error) {
	trimmed := strings.TrimSpace(idOrSlug)
	if trimmed == "" {
		return domain.PricingModel{}, repositories.ErrNotFound
	}

	if id, err := uuid.Parse(trimmed); err == nil {
		m, err := s.pricingModels.Get(ctx, orgID, id)
		if err == nil {
			return m, nil
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			return domain.PricingModel{}, err
		}
	}

	m, err := s.pricingModels.GetBySlug(ctx, orgID, trimmed)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.PricingModel{}, repositories.ErrNotFound
		}
		return domain.PricingModel{}, err
	}
	return m, nil
}
