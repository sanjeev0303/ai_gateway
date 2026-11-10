package gatewayv2

import (
	"context"
	"sync"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

const modelCatalogTTL = 30 * time.Second

// modelCatalog answers which provider serves a model and whether it may be
// called. It keeps one copy of the model registry, refreshed every 30 seconds
// or when the registry changes, instead of scanning the table per request.
// Models the registry has never seen fall back to the name rules in
// domain.ProviderForModel.
type modelCatalog struct {
	repo repositories.ModelRepository

	mu       sync.Mutex
	loaded   bool
	loadedAt time.Time
	models   []domain.Model
}

func newModelCatalog(repo repositories.ModelRepository) *modelCatalog {
	return &modelCatalog{repo: repo}
}

// Invalidate drops the cached registry so the next lookup reloads it.
func (c *modelCatalog) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.loaded = false
	c.mu.Unlock()
}

func (c *modelCatalog) registry(ctx context.Context) []domain.Model {
	if c == nil || c.repo == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded && time.Since(c.loadedAt) < modelCatalogTTL {
		return c.models
	}
	// Inactive models are kept too, so a disabled model is refused rather
	// than waved through by its name.
	models, err := c.repo.ListAll(ctx)
	if err != nil {
		return c.models // keep serving the last good copy
	}
	c.models = models
	c.loaded = true
	c.loadedAt = time.Now()
	return models
}

// ProviderFor returns the provider for a model: from its name when that says,
// otherwise from the registry (Gemini's "gemma-" models, for one).
func (c *modelCatalog) ProviderFor(ctx context.Context, model string) string {
	if p := domain.ProviderForModel(model); p != "" {
		return p
	}
	for _, m := range c.registry(ctx) {
		if m.ModelSlug == model && m.IsActive {
			return m.ProviderKind
		}
	}
	return ""
}

// Supported reports whether a model may be called through providerKind. A
// registry entry decides when there is one; otherwise any recognised family,
// or any model on a custom connection, is allowed.
func (c *modelCatalog) Supported(ctx context.Context, providerKind, model string) bool {
	if model == "" {
		return true
	}
	models := c.registry(ctx)
	if m, ok := findModel(models, providerKind, func(m domain.Model) bool { return m.ModelSlug == model }); ok {
		return m.IsActive && m.Status != "inactive"
	}
	if m, ok := findModel(models, providerKind, func(m domain.Model) bool { return m.DisplayName == model }); ok {
		return m.IsActive && m.Status != "inactive"
	}
	return domain.ProviderForModel(model) != "" || providerKind == domain.ProviderKindCustom
}

func findModel(models []domain.Model, providerKind string, match func(domain.Model) bool) (domain.Model, bool) {
	for _, m := range models {
		if match(m) && (providerKind == "" || m.ProviderKind == providerKind) {
			return m, true
		}
	}
	return domain.Model{}, false
}
