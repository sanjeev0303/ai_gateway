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

type stubInferenceProvider struct {
	stubProvider
	meta providerclient.ProviderMetadata
	caps providerclient.ModelCapabilities
}

func (s *stubInferenceProvider) Name() string { return "Stub Inference" }
func (s *stubInferenceProvider) Kind() string { return domain.ProviderKindOpenAI }
func (s *stubInferenceProvider) Metadata() providerclient.ProviderMetadata { return s.meta }
func (s *stubInferenceProvider) GetModelCapabilities(_ context.Context, modelID string) (providerclient.ModelCapabilities, error) {
	caps := s.caps
	caps.ModelID = modelID
	return caps, nil
}
func (s *stubInferenceProvider) Execute(_ context.Context, _ string, _ providerclient.NormalizedRequest) (*providerclient.NormalizedResponse, error) {
	return nil, nil
}
func (s *stubInferenceProvider) Stream(_ context.Context, _ string, _ providerclient.NormalizedRequest) (providerclient.NormalizedStream, error) {
	return nil, nil
}
func (s *stubInferenceProvider) EstimateTokens(_ context.Context, _ string, _ providerclient.NormalizedRequest) (*providerclient.TokenUsage, error) {
	return nil, nil
}

func TestRegistryInferenceProviderResolution(t *testing.T) {
	ctx := context.Background()
	plainStub := &stubProvider{}
	infStub := &stubInferenceProvider{
		meta: providerclient.ProviderMetadata{
			Kind:        domain.ProviderKindOpenAI,
			DisplayName: "Stub OpenAI",
		},
		caps: providerclient.ModelCapabilities{
			Provider:     domain.ProviderKindOpenAI,
			ContextLimit: 128000,
		},
	}

	r := NewRegistry(map[string]providerclient.Provider{
		domain.ProviderKindAnthropic: plainStub,
		domain.ProviderKindOpenAI:    infStub,
	})

	// Plain provider does not implement InferenceProvider
	if ip := r.InferenceProvider(domain.ProviderKindAnthropic); ip != nil {
		t.Errorf("expected nil InferenceProvider for plain stub, got %v", ip)
	}
	if meta := r.Metadata(domain.ProviderKindAnthropic); meta != nil {
		t.Errorf("expected nil Metadata for plain stub, got %v", meta)
	}

	// Registered InferenceProvider resolves
	ip := r.InferenceProvider(domain.ProviderKindOpenAI)
	if ip == nil {
		t.Fatalf("expected non-nil InferenceProvider for infStub")
	}
	if ip.Name() != "Stub Inference" {
		t.Errorf("expected name 'Stub Inference', got %s", ip.Name())
	}

	meta := r.Metadata(domain.ProviderKindOpenAI)
	if meta == nil || meta.DisplayName != "Stub OpenAI" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	caps, err := r.GetModelCapabilities(ctx, domain.ProviderKindOpenAI, "gpt-4o")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if caps.ModelID != "gpt-4o" || caps.ContextLimit != 128000 {
		t.Errorf("unexpected capabilities: %+v", caps)
	}

	// Unknown provider returns ErrUnknownProvider
	_, err = r.GetModelCapabilities(ctx, "nonexistent", "model-1")
	if !errors.Is(err, providerclient.ErrUnknownProvider) {
		t.Errorf("expected ErrUnknownProvider, got %v", err)
	}
}

