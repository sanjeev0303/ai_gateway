package provider

import (
	"context"
	"errors"
	"fmt"

	"ai-governance-saas/backend/internal/domain"
)

// Resolution errors, following the "provider: ..." sentinel style of
// ErrInvalidProviderCredential in service.go.
var (
	// ErrProviderNotRegistered is returned when the requested provider kind
	// has no client in the registry.
	ErrProviderNotRegistered = errors.New("provider: provider kind not registered")
	// ErrCredentialMissing is returned when no API key is supplied for a
	// configured provider, so its models cannot be listed.
	ErrCredentialMissing = errors.New("provider: no api key for provider kind")
	// ErrModelNotFound is returned when the preferred provider does not offer
	// the requested model and fallback cannot resolve it.
	ErrModelNotFound = errors.New("provider: model not offered by provider")
)

// ResolveRequest carries the caller's preferred provider, model, and region.
// API keys are passed per call because they are tenant-scoped and resolved by
// the caller (credentials/secrets store lookup), never held by the resolver.
type ResolveRequest struct {
	ProviderKind string
	ModelID      string
	Region       string
}

// ModelHandle is the resolved routing target for a model request. ModelID is
// the provider-qualified slug (see QualifiedModelID); FallbackUsed reports
// whether the target came from the FallbackPolicy rather than the request.
type ModelHandle struct {
	ProviderKind string
	ModelID      string
	Region       string
	FallbackUsed bool
}

// ModelResolver maps a provider+model+region request to the registered
// provider client that should serve it, applying the FallbackPolicy when the
// preferred provider does not offer the requested model.
type ModelResolver struct {
	registry *Registry
	fallback *domain.FallbackPolicy
}

// NewModelResolver wires the resolver from the registry and fallback policy.
// A nil or zero-value FallbackPolicy disables fallback (opt-in).
func NewModelResolver(registry *Registry, fallback *domain.FallbackPolicy) *ModelResolver {
	return &ModelResolver{registry: registry, fallback: fallback}
}

// Resolve resolves the request to a handle. The preferred provider must be
// registered and have an API key in keys. Fallback applies only when the
// preferred provider does not offer the requested model (error-only): it
// never cascades — the fallback target is returned as-is and is never itself
// re-resolved, and provider-level failures (credential missing, listing
// error) never trigger fallback.
func (r *ModelResolver) Resolve(ctx context.Context, keys map[string]string, req ResolveRequest) (ModelHandle, error) {
	if r.registry.Provider(req.ProviderKind) == nil {
		return ModelHandle{}, fmt.Errorf("%w: %q", ErrProviderNotRegistered, req.ProviderKind)
	}
	if keys[req.ProviderKind] == "" {
		return ModelHandle{}, fmt.Errorf("%w: %q", ErrCredentialMissing, req.ProviderKind)
	}
	offered, err := r.listModels(ctx, req.ProviderKind, keys[req.ProviderKind])
	if err != nil {
		return ModelHandle{}, err
	}
	if offered[req.ModelID] {
		return ModelHandle{
			ProviderKind: req.ProviderKind,
			ModelID:      QualifiedModelID(req.ProviderKind, req.ModelID),
			Region:       req.Region,
		}, nil
	}

	if r.fallback == nil || r.fallback.FallbackProvider == "" {
		return ModelHandle{}, fmt.Errorf("%w: %q on %q", ErrModelNotFound, req.ModelID, req.ProviderKind)
	}
	if err := r.fallback.Validate(); err != nil {
		return ModelHandle{}, err
	}
	if r.fallback.FallbackProvider == req.ProviderKind {
		return ModelHandle{}, fmt.Errorf("%w: fallback provider %q equals preferred", ErrModelNotFound, req.ProviderKind)
	}
	if r.registry.Provider(r.fallback.FallbackProvider) == nil {
		return ModelHandle{}, fmt.Errorf("%w: fallback provider %q not registered", ErrModelNotFound, r.fallback.FallbackProvider)
	}
	if keys[r.fallback.FallbackProvider] == "" {
		return ModelHandle{}, fmt.Errorf("%w: no key for fallback provider %q", ErrModelNotFound, r.fallback.FallbackProvider)
	}
	fallbackOffered, err := r.listModels(ctx, r.fallback.FallbackProvider, keys[r.fallback.FallbackProvider])
	if err != nil {
		return ModelHandle{}, err
	}
	if !fallbackOffered[r.fallback.FallbackModel] {
		return ModelHandle{}, fmt.Errorf("%w: %q on %q (fallback %q does not offer it)",
			ErrModelNotFound, req.ModelID, req.ProviderKind, r.fallback.FallbackProvider)
	}
	return ModelHandle{
		ProviderKind: r.fallback.FallbackProvider,
		ModelID:      QualifiedModelID(r.fallback.FallbackProvider, r.fallback.FallbackModel),
		Region:       req.Region,
		FallbackUsed: true,
	}, nil
}

// listModels returns the set of model IDs the provider advertises, via its
// registered client. Listing errors are surfaced as-is (never swallowed).
func (r *ModelResolver) listModels(ctx context.Context, kind, apiKey string) (map[string]bool, error) {
	client := r.registry.Provider(kind)
	models, err := client.ListModels(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(models))
	for _, m := range models {
		out[m.ID] = true
	}
	return out, nil
}
