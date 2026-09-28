// Package openai implements the OpenAI provider adapter.
package openai

import (
	"context"

	"ai-governance-saas/backend/internal/ports/provider"
)

// ProviderAdapter wraps the OpenAI client and implements the provider.Provider interface.
type ProviderAdapter struct {
	baseURL string
	version string
}

// NewProviderAdapter creates a new OpenAI provider adapter.
func NewProviderAdapter() *ProviderAdapter {
	return &ProviderAdapter{
		baseURL: defaultBaseURL,
		version: "2024-08-01-preview",
	}
}

// ValidateCredentials verifies the API key against OpenAI.
func (a *ProviderAdapter) ValidateCredentials(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return err
	}
	return client.ValidateCredentials(ctx, apiKey)
}

// HealthCheck proves connectivity to OpenAI.
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
	return client.ListModels(ctx, apiKey)
}

// ---- Extended API for gateway integration ------------------------------------

// CreateMessage sends a non-streaming message request.
func (a *ProviderAdapter) CreateMessage(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessage(ctx, apiKey, req)
}

// CreateMessageStream sends a streaming message request.
func (a *ProviderAdapter) CreateMessageStream(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*MessageStream, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessageStream(ctx, apiKey, req)
}

// CountTokens counts tokens for a request.
func (a *ProviderAdapter) CountTokens(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*CountTokensResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CountTokens(ctx, apiKey, req)
}
