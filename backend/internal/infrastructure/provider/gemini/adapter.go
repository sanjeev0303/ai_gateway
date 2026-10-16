// Package gemini implements the Gemini provider adapter.
package gemini

import (
	"context"
	"errors"
	"fmt"

	"ai-governance-saas/backend/internal/ports/provider"
)

// ProviderAdapter wraps the Gemini client and implements the provider.Provider interface.
type ProviderAdapter struct {
	baseURL string
	version string
}

// NewProviderAdapter creates a new Gemini provider adapter.
func NewProviderAdapter() *ProviderAdapter {
	return &ProviderAdapter{
		baseURL: defaultBaseURL,
		version: defaultVersion,
	}
}

// ValidateCredentials verifies the API key against Gemini.
func (a *ProviderAdapter) ValidateCredentials(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return err
	}
	if err := client.ValidateCredentials(ctx, apiKey); err != nil {
		if errors.Is(err, ErrInvalidAPIKey) {
			return fmt.Errorf("%w: %w", ErrInvalidAPIKey, provider.ErrInvalidCredentials)
		}
		return err
	}
	return nil
}

// HealthCheck proves connectivity to Gemini.
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
func (a *ProviderAdapter) CreateMessage(ctx context.Context, apiKey string, req *GenerateContentRequest) (*GenerateContentResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessage(ctx, apiKey, req)
}

// CreateMessageStream sends a streaming message request.
func (a *ProviderAdapter) CreateMessageStream(ctx context.Context, apiKey string, req *GenerateContentRequest) (*MessageStream, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CreateMessageStream(ctx, apiKey, req)
}

// CountTokens counts tokens for a request.
func (a *ProviderAdapter) CountTokens(ctx context.Context, apiKey string, model string, req *CountTokensRequest) (*CountTokensResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway")
	if err != nil {
		return nil, err
	}
	return client.CountTokens(ctx, apiKey, model, req)
}
