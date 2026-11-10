package gatewayv2

import (
	"context"
	"testing"

	"ai-governance-saas/backend/internal/domain"
)

func TestModelCatalog_ProviderForUnprefixedModel(t *testing.T) {
	c := newModelCatalog(&stubMultiOrgModelRepo{models: []domain.Model{
		{ProviderKind: domain.ProviderKindGemini, ModelSlug: "gemma-4-31b-it", IsActive: true, Status: "active"},
		{ProviderKind: domain.ProviderKindGemini, ModelSlug: "gemma-retired", IsActive: false, Status: "inactive"},
	}})

	if got := c.ProviderFor(context.Background(), "gemma-4-31b-it"); got != domain.ProviderKindGemini {
		t.Errorf("gemma-4-31b-it provider = %q, want gemini", got)
	}
	if got := c.ProviderFor(context.Background(), "gemma-retired"); got != "" {
		t.Errorf("inactive model resolved to %q, want none", got)
	}
	if got := c.ProviderFor(context.Background(), "unknown-model"); got != "" {
		t.Errorf("unknown model resolved to %q, want none", got)
	}
	if got := newModelCatalog(nil).ProviderFor(context.Background(), "gemma-4-31b-it"); got != "" {
		t.Errorf("catalog without a registry resolved %q", got)
	}
}

// A disabled registry entry is refused even when its name looks like a known
// family; an unknown name of a known family is allowed.
func TestModelCatalog_Supported(t *testing.T) {
	c := newModelCatalog(&stubMultiOrgModelRepo{models: []domain.Model{
		{ProviderKind: domain.ProviderKindOpenAI, ModelSlug: "gpt-4o", IsActive: true, Status: "active"},
		{ProviderKind: domain.ProviderKindOpenAI, ModelSlug: "gpt-3.5-turbo", IsActive: false, Status: "inactive"},
	}})
	ctx := context.Background()

	if !c.Supported(ctx, domain.ProviderKindOpenAI, "gpt-4o") {
		t.Error("active registry model refused")
	}
	if c.Supported(ctx, domain.ProviderKindOpenAI, "gpt-3.5-turbo") {
		t.Error("disabled registry model allowed")
	}
	if !c.Supported(ctx, domain.ProviderKindOpenAI, "gpt-5") {
		t.Error("known family refused")
	}
	if c.Supported(ctx, domain.ProviderKindOpenAI, "made-up-model") {
		t.Error("unknown model allowed")
	}
}

// The registry is read once and served from cache until invalidated.
func TestModelCatalog_CachesRegistry(t *testing.T) {
	repo := &countingModelRepo{stubMultiOrgModelRepo: stubMultiOrgModelRepo{models: []domain.Model{
		{ProviderKind: domain.ProviderKindGemini, ModelSlug: "gemma-4-31b-it", IsActive: true, Status: "active"},
	}}}
	c := newModelCatalog(repo)
	for i := 0; i < 5; i++ {
		c.ProviderFor(context.Background(), "gemma-4-31b-it")
	}
	if repo.lists != 1 {
		t.Fatalf("registry read %d times for 5 lookups, want 1", repo.lists)
	}
	c.Invalidate()
	c.ProviderFor(context.Background(), "gemma-4-31b-it")
	if repo.lists != 2 {
		t.Fatalf("invalidate did not force a reload (%d reads)", repo.lists)
	}
}

type countingModelRepo struct {
	stubMultiOrgModelRepo
	lists int
}

func (r *countingModelRepo) ListAll(ctx context.Context) ([]domain.Model, error) {
	r.lists++
	return r.stubMultiOrgModelRepo.ListAll(ctx)
}
