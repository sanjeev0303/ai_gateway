package provider

import (
	"context"
	"sort"

	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

// Registry is the application-layer set of registered provider clients,
// keyed by provider kind. Model-catalog seeding and the gateway router both
// draw from it: registering a provider under its kind makes its models
// discoverable through ListModels.
type Registry struct {
	providers map[string]providerclient.Provider
}

// NewRegistry wires the registry from its provider clients. The map is
// copied, so later mutation by the caller does not affect the registry.
func NewRegistry(providers map[string]providerclient.Provider) *Registry {
	cp := make(map[string]providerclient.Provider, len(providers))
	for kind, p := range providers {
		cp[kind] = p
	}
	return &Registry{providers: cp}
}

// Provider returns the client registered under kind, or nil when absent.
func (r *Registry) Provider(kind string) providerclient.Provider {
	return r.providers[kind]
}

// ListModels aggregates the provider-advertised models of every registered
// provider whose key is present in apiKeys, in stable kind-then-ID order. A
// provider without a key is skipped so one missing credential does not fail
// the whole listing. Each entry carries the Provider tag its adapter set;
// use QualifiedModelID for the catalog's provider-prefixed slug.
func (r *Registry) ListModels(ctx context.Context, apiKeys map[string]string) ([]providerclient.Model, error) {
	kinds := make([]string, 0, len(r.providers))
	for kind := range r.providers {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	var out []providerclient.Model
	for _, kind := range kinds {
		key, ok := apiKeys[kind]
		if !ok {
			continue
		}
		models, err := r.providers[kind].ListModels(ctx, key)
		if err != nil {
			return nil, err
		}
		sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
		out = append(out, models...)
	}
	return out, nil
}

// QualifiedModelID returns the catalog slug for a model: the provider kind
// and the wire ID joined by a slash (e.g. "anthropic/claude-sonnet-4-5").
func QualifiedModelID(kind, modelID string) string {
	return kind + "/" + modelID
}
