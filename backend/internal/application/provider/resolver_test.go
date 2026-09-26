package provider

import (
	"context"
	"errors"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

func modelsProvider(t *testing.T, ids ...string) *stubProvider {
	t.Helper()
	return &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		out := make([]providerclient.Model, 0, len(ids))
		for _, id := range ids {
			out = append(out, providerclient.Model{ID: id})
		}
		return out, nil
	}}
}

func newResolver(t *testing.T, fallback *domain.FallbackPolicy, providers map[string]providerclient.Provider) *ModelResolver {
	t.Helper()
	return NewModelResolver(NewRegistry(providers), fallback)
}

func baseKeys() map[string]string {
	return map[string]string{
		domain.ProviderKindAnthropic: "sk-ant-test",
		domain.ProviderKindOpenAI:    "sk-op-test",
		domain.ProviderKindGemini:    "gem-test",
	}
}

func TestResolverPreferredProvider(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, nil, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
		domain.ProviderKindOpenAI:    modelsProvider(t, "gpt-4o"),
	})

	h, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "claude-sonnet-4-5",
		Region:       "us-east",
	})
	requireNoErr(t, err)
	if h.ProviderKind != domain.ProviderKindAnthropic {
		t.Fatalf("ProviderKind = %q, want anthropic", h.ProviderKind)
	}
	if h.ModelID != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("ModelID = %q, want qualified slug", h.ModelID)
	}
	if h.Region != "us-east" {
		t.Fatalf("Region = %q, want us-east", h.Region)
	}
	if h.FallbackUsed {
		t.Fatal("FallbackUsed = true, want false")
	}
}

func TestResolverProviderNotRegistered(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, nil, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindGemini,
		ModelID:      "gemini-2.5-pro",
	})
	requireErrIs(t, err, ErrProviderNotRegistered)
}

func TestResolverCredentialMissing(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, nil, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	keys := map[string]string{} // no key for anthropic
	_, err := r.Resolve(ctx, keys, ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "claude-sonnet-4-5",
	})
	requireErrIs(t, err, ErrCredentialMissing)
}

func TestResolverModelNotFoundWithoutFallback(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, nil, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "claude-opus-4-1",
	})
	requireErrIs(t, err, ErrModelNotFound)
}

func TestResolverFallbackPicksAlternateProvider(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: domain.ProviderKindOpenAI,
		FallbackModel:    "gpt-4o",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"), // no gpt-4o
		domain.ProviderKindOpenAI:    modelsProvider(t, "gpt-4o"),
	})

	h, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "gpt-4o", // anthropic does not offer it
	})
	requireNoErr(t, err)
	if !h.FallbackUsed {
		t.Fatal("FallbackUsed = false, want true")
	}
	if h.ProviderKind != domain.ProviderKindOpenAI {
		t.Fatalf("ProviderKind = %q, want openai", h.ProviderKind)
	}
	if h.ModelID != "openai/gpt-4o" {
		t.Fatalf("ModelID = %q, want openai/gpt-4o", h.ModelID)
	}
}

func TestResolverFallbackNeverCascades(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: domain.ProviderKindOpenAI,
		FallbackModel:    "gpt-4o",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
		domain.ProviderKindOpenAI:    modelsProvider(t, "claude-sonnet-4-5"), // fallback does NOT offer gpt-4o
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "gpt-4o",
	})
	requireErrIs(t, err, ErrModelNotFound)
}

func TestResolverFallbackProviderNotRegistered(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: domain.ProviderKindGemini, // not in registry
		FallbackModel:    "gemini-2.5-pro",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "gpt-4o",
	})
	requireErrIs(t, err, ErrModelNotFound)
}

func TestResolverFallbackPolicyInvalid(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: "not-a-provider",
		FallbackModel:    "gpt-4o",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "gpt-4o",
	})
	if err == nil {
		t.Fatal("Resolve() = nil, want invalid fallback policy error")
	}
	if errors.Is(err, ErrModelNotFound) {
		t.Fatalf("Resolve() = %v, want policy validation error, not ErrModelNotFound", err)
	}
}

func TestResolverFallbackSameAsPreferred(t *testing.T) {
	ctx := context.Background()
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: domain.ProviderKindAnthropic, // == preferred
		FallbackModel:    "gpt-4o",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: modelsProvider(t, "claude-sonnet-4-5"),
	})

	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "gpt-4o",
	})
	requireErrIs(t, err, ErrModelNotFound)
}

func TestResolverListingErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("upstream down")
	client := &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		return nil, wantErr
	}}
	r := newResolver(t, nil, map[string]providerclient.Provider{
		domain.ProviderKindOpenAI: client,
	})

	// A listing failure is provider-level and must surface as-is; fallback is
	// error-only for model-missing and must NOT mask upstream failures.
	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindOpenAI,
		ModelID:      "gpt-4o",
	})
	requireErrIs(t, err, wantErr)
}

func TestResolverListingErrorNeverFallback(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("upstream down")
	broken := &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		return nil, wantErr
	}}
	r := newResolver(t, &domain.FallbackPolicy{
		FallbackProvider: domain.ProviderKindGemini,
		FallbackModel:    "gemini-2.5-pro",
	}, map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: broken, // preferred is down
		domain.ProviderKindGemini:    modelsProvider(t, "gemini-2.5-pro"),
	})

	// Fallback applies only when the model is missing, never on an upstream
	// listing error from the preferred provider.
	_, err := r.Resolve(ctx, baseKeys(), ResolveRequest{
		ProviderKind: domain.ProviderKindAnthropic,
		ModelID:      "claude-sonnet-4-5",
	})
	requireErrIs(t, err, wantErr)
}