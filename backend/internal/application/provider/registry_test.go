package provider

import (
	"context"
	"errors"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

// Provider registry: client lookup by kind and cross-provider model listing.

func TestRegistryProviderLookup(t *testing.T) {
	client := &stubProvider{}
	r := NewRegistry(map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: client,
	})
	if got := r.Provider(domain.ProviderKindAnthropic); got != client {
		t.Fatal("registered kind did not resolve to its client")
	}
	if got := r.Provider(domain.ProviderKindOpenAI); got != nil {
		t.Fatalf("unregistered kind resolved to %v, want nil", got)
	}
}

func TestRegistryListModelsAggregatesSorted(t *testing.T) {
	ctx := context.Background()
	anthropic := &stubProvider{listModels: func(_ context.Context, key string) ([]providerclient.Model, error) {
		if key != "sk-ant-1" {
			t.Fatalf("anthropic got key %q, want sk-ant-1", key)
		}
		return []providerclient.Model{
			{ID: "claude-sonnet-4-5", Provider: domain.ProviderKindAnthropic},
			{ID: "claude-opus-4-1", Provider: domain.ProviderKindAnthropic},
		}, nil
	}}
	openai := &stubProvider{listModels: func(_ context.Context, key string) ([]providerclient.Model, error) {
		if key != "sk-openai-1" {
			t.Fatalf("openai got key %q, want sk-openai-1", key)
		}
		return []providerclient.Model{{ID: "gpt-4o", Provider: domain.ProviderKindOpenAI}}, nil
	}}
	r := NewRegistry(map[string]providerclient.Provider{
		domain.ProviderKindOpenAI:    openai,
		domain.ProviderKindAnthropic: anthropic,
	})

	models, err := r.ListModels(ctx, map[string]string{
		domain.ProviderKindAnthropic: "sk-ant-1",
		domain.ProviderKindOpenAI:    "sk-openai-1",
	})
	requireNoErr(t, err)
	if len(models) != 3 {
		t.Fatalf("got %d models, want 3", len(models))
	}
	// Stable order: anthropic (ids sorted) before openai.
	want := []string{"claude-opus-4-1", "claude-sonnet-4-5", "gpt-4o"}
	for i, m := range models {
		if m.ID != want[i] {
			t.Fatalf("models[%d].ID = %q, want %q", i, m.ID, want[i])
		}
		if m.Provider == "" {
			t.Fatalf("models[%d] missing Provider tag", i)
		}
	}
}

func TestRegistryListModelsSkipsProviderWithoutKey(t *testing.T) {
	ctx := context.Background()
	called := false
	openai := &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		called = true
		return []providerclient.Model{{ID: "gpt-4o", Provider: domain.ProviderKindOpenAI}}, nil
	}}
	anthropic := &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		t.Fatal("anthropic listed without a key")
		return nil, nil
	}}
	r := NewRegistry(map[string]providerclient.Provider{
		domain.ProviderKindOpenAI:    openai,
		domain.ProviderKindAnthropic: anthropic,
	})

	models, err := r.ListModels(ctx, map[string]string{domain.ProviderKindOpenAI: "sk-1"})
	requireNoErr(t, err)
	if !called {
		t.Fatal("openai ListModels was not called")
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
}

func TestRegistryListModelsPropagatesError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom")
	openai := &stubProvider{listModels: func(context.Context, string) ([]providerclient.Model, error) {
		return nil, wantErr
	}}
	r := NewRegistry(map[string]providerclient.Provider{
		domain.ProviderKindOpenAI: openai,
	})

	_, err := r.ListModels(ctx, map[string]string{domain.ProviderKindOpenAI: "sk-1"})
	requireErrIs(t, err, wantErr)
}

func TestQualifiedModelID(t *testing.T) {
	if got := QualifiedModelID(domain.ProviderKindAnthropic, "claude-sonnet-4-5"); got != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("QualifiedModelID = %q", got)
	}
}