package gatewayv2

import (
	"context"
	"testing"

	"ai-governance-saas/backend/internal/domain"
)

func TestRegistryProviderForUnprefixedModel(t *testing.T) {
	g := &Gateway{modelRepo: &stubMultiOrgModelRepo{models: []domain.Model{
		{ProviderKind: domain.ProviderKindGemini, ModelSlug: "gemma-4-31b-it", IsActive: true, Status: "active"},
		{ProviderKind: domain.ProviderKindGemini, ModelSlug: "gemma-retired", IsActive: false, Status: "inactive"},
	}}}

	if got := g.registryProviderFor(context.Background(), "gemma-4-31b-it"); got != domain.ProviderKindGemini {
		t.Errorf("gemma-4-31b-it provider = %q, want gemini", got)
	}
	if got := g.registryProviderFor(context.Background(), "gemma-retired"); got != "" {
		t.Errorf("inactive model resolved to %q, want none", got)
	}
	if got := g.registryProviderFor(context.Background(), "unknown-model"); got != "" {
		t.Errorf("unknown model resolved to %q, want none", got)
	}
	if got := (&Gateway{}).registryProviderFor(context.Background(), "gemma-4-31b-it"); got != "" {
		t.Errorf("gateway without a registry resolved %q", got)
	}
}
