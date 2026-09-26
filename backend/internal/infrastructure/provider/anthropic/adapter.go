// Package anthropic implements the Anthropic provider adapter.
package anthropic

import (
	"context"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

// ProviderAdapter wraps the Anthropic client and implements the provider.Provider interface.
type ProviderAdapter struct {
	baseURL string
	version string
}

// NewProviderAdapter creates a new Anthropic provider adapter.
func NewProviderAdapter() *ProviderAdapter {
	return &ProviderAdapter{
		baseURL: defaultBaseURL,
		version: defaultAnthropicVersion,
	}
}

// ValidateCredentials verifies the API key against Anthropic.
func (a *ProviderAdapter) ValidateCredentials(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return err
	}
	return client.ValidateCredentials(ctx, apiKey)
}

// HealthCheck proves connectivity to Anthropic.
func (a *ProviderAdapter) HealthCheck(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return err
	}
	return client.HealthCheck(ctx, apiKey)
}

// ListModels returns the provider's model IDs.
func (a *ProviderAdapter) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	localModels, err := client.ListModels(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	models := make([]provider.Model, len(localModels))
	for i, m := range localModels {
		models[i] = provider.Model{
			ID:          m.ID,
			Provider:    domain.ProviderKindAnthropic,
			DisplayName: m.DisplayName,
		}
	}
	return models, nil
}

// ---- Extended API for gateway integration ------------------------------------

// CreateMessage sends a non-streaming message request.
func (a *ProviderAdapter) CreateMessage(ctx context.Context, apiKey string, req *MessageRequest) (*MessageResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessage(ctx, apiKey, req)
}

// CreateMessageStream sends a streaming message request.
func (a *ProviderAdapter) CreateMessageStream(ctx context.Context, apiKey string, req *MessageRequest) (*MessageStream, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessageStream(ctx, apiKey, req)
}

// CountTokens counts tokens for a request.
func (a *ProviderAdapter) CountTokens(ctx context.Context, apiKey string, req *CountTokensRequest) (*CountTokensResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CountTokens(ctx, apiKey, req)
}