// Package openai implements the OpenAI provider adapter.
package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-governance-saas/backend/internal/ports/provider"
)

// ProviderAdapter wraps the OpenAI client and implements the provider.Provider interface.
type ProviderAdapter struct {
	baseURL string
	version string
	// allowPrivate disables the SSRF dial guard on the per-account client. The
	// account base URL is tenant-controlled, so this is false outside local dev.
	allowPrivate bool
}

// NewProviderAdapter creates a new OpenAI provider adapter.
func NewProviderAdapter(allowPrivate bool) *ProviderAdapter {
	return &ProviderAdapter{
		baseURL:      defaultBaseURL,
		version:      "2024-08-01-preview",
		allowPrivate: allowPrivate,
	}
}

// NewCustomProviderAdapter creates an OpenAI provider adapter with custom base URL and version.
func NewCustomProviderAdapter(baseURL, version string, allowPrivate bool) *ProviderAdapter {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if version == "" {
		version = "2024-08-01-preview"
	}
	return &ProviderAdapter{
		baseURL:      baseURL,
		version:      version,
		allowPrivate: allowPrivate,
	}
}

// WithBaseURL returns an adapter for an account's own endpoint. The client adds
// "/v1/..." itself, so a trailing "/v1" in the account URL is trimmed.
func (a *ProviderAdapter) WithBaseURL(baseURL string) provider.Provider {
	u := strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(baseURL), "/"), "/v1")
	if u == "" {
		return a
	}
	return &ProviderAdapter{baseURL: u, version: a.version, allowPrivate: a.allowPrivate}
}

// ValidateCredentials verifies the API key against OpenAI.
func (a *ProviderAdapter) ValidateCredentials(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
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

// HealthCheck proves connectivity to OpenAI.
func (a *ProviderAdapter) HealthCheck(ctx context.Context, apiKey string) error {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
	if err != nil {
		return err
	}
	return client.HealthCheck(ctx, apiKey)
}

// ListModels returns the provider's model IDs.
func (a *ProviderAdapter) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
	if err != nil {
		return nil, err
	}
	models, err := client.ListModels(ctx, apiKey)
	if err != nil {
		if errors.Is(err, ErrInvalidAPIKey) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidAPIKey, provider.ErrInvalidCredentials)
		}
		return nil, err
	}
	return models, nil
}

// ---- Extended API for gateway integration ------------------------------------

// CreateMessage sends a non-streaming message request.
func (a *ProviderAdapter) CreateMessage(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
	if err != nil {
		return nil, err
	}
	return client.CreateMessage(ctx, apiKey, req)
}

// CreateMessageStream sends a streaming message request.
func (a *ProviderAdapter) CreateMessageStream(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*MessageStream, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
	if err != nil {
		return nil, err
	}
	return client.CreateMessageStream(ctx, apiKey, req)
}

// CountTokens counts tokens for a request.
func (a *ProviderAdapter) CountTokens(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*CountTokensResponse, error) {
	client, err := New(a.baseURL, a.version, "ai-gateway", a.allowPrivate)
	if err != nil {
		return nil, err
	}
	return client.CountTokens(ctx, apiKey, req)
}
