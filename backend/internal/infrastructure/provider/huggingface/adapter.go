// Package huggingface implements the Hugging Face provider adapter.
package huggingface

import (
	"context"
	"errors"
	"fmt"

	"ai-governance-saas/backend/internal/ports/provider"
)

// ProviderAdapter wraps the Hugging Face client and implements the provider.Provider interface.
type ProviderAdapter struct {
	baseURL string
	hubURL  string
}

// NewProviderAdapter creates a new Hugging Face provider adapter with defaults.
func NewProviderAdapter() *ProviderAdapter {
	return &ProviderAdapter{
		baseURL: defaultBaseURL,
		hubURL:  defaultHubURL,
	}
}

// NewCustomProviderAdapter creates an adapter with custom endpoints.
func NewCustomProviderAdapter(baseURL, hubURL string) *ProviderAdapter {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if hubURL == "" {
		hubURL = defaultHubURL
	}
	return &ProviderAdapter{
		baseURL: baseURL,
		hubURL:  hubURL,
	}
}

// ValidateCredentials verifies the API key against Hugging Face.
func (a *ProviderAdapter) ValidateCredentials(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return err
	}
	if err := client.ValidateCredentials(ctx, apiKey); err != nil {
		if errors.Is(err, ErrInvalidAPIKey) || errors.Is(err, provider.ErrInvalidCredentials) {
			return fmt.Errorf("%w: %w", ErrInvalidAPIKey, provider.ErrInvalidCredentials)
		}
		return err
	}
	return nil
}

// HealthCheck proves connectivity to Hugging Face.
func (a *ProviderAdapter) HealthCheck(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return err
	}
	return client.HealthCheck(ctx, apiKey)
}

// ListModels returns Hugging Face models.
func (a *ProviderAdapter) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return nil, err
	}
	return client.ListModels(ctx, apiKey)
}

// ---- Extended API for gateway integration ------------------------------------

// CreateMessage sends a non-streaming chat request.
func (a *ProviderAdapter) CreateMessage(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessage(ctx, apiKey, req)
}

// CreateMessageStream sends a streaming chat request.
func (a *ProviderAdapter) CreateMessageStream(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*MessageStream, error) {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessageStream(ctx, apiKey, req)
}

// CountTokens counts or estimates tokens for a request.
func (a *ProviderAdapter) CountTokens(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*CountTokensResponse, error) {
	client, err := New(a.baseURL, a.hubURL, "Sentinel-AI-Gateway")
	if err != nil {
		return nil, err
	}
	return client.CountTokens(ctx, apiKey, req)
}
