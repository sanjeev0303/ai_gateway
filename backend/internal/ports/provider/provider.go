// Package provider defines the provider-client port (M4). M4 implements
// credential validation, connectivity health checks and the model catalog;
// request streaming lands as a byte-level reverse proxy in the gateway, which
// is transport and therefore not part of this port. Payloads pass through
// provider-native (Claude Code included) — this port never rewrites them.
package provider

import (
	"context"
	"errors"
)

// ErrInvalidCredentials marks a provider rejection of an API key (HTTP 401
// or 403). Callers must treat this as a non-retryable, user-facing error.
var ErrInvalidCredentials = errors.New("provider: invalid credentials")

// Provider talks to one upstream LLM provider (Anthropic, OpenAI, Gemini, …).
// Implementations must never log API keys or request payloads.
type Provider interface {
	// ValidateCredentials verifies the API key against the provider. It
	// returns ErrInvalidCredentials on 401/403, and a transport/protocol
	// error otherwise.
	ValidateCredentials(ctx context.Context, apiKey string) error
	// HealthCheck proves connectivity to the provider. It succeeds on any
	// provider response (including 401, which proves reachability) and
	// fails only on transport-level errors.
	HealthCheck(ctx context.Context, apiKey string) error
	// ListModels returns the provider's model IDs.
	ListModels(ctx context.Context, apiKey string) ([]Model, error)
}

// Model is one model the provider exposes, with the metadata the
// provider's catalog API advertises. Fields a provider does not expose
// are left at their zero value; the model registry layers
// governance-owned fields (status, pricing_version, activation flags)
// on top when seeding the DB catalog. Providers must never fabricate
// metadata the upstream API does not return.
type Model struct {
	// ID is the provider's model identifier as used on the wire (e.g.
	// "claude-sonnet-4-5", "gpt-4o", "gemini-2.5-flash").
	ID string `json:"id"`

	// Provider is the provider kind this model belongs to, one of the
	// domain.ProviderKind* values ("openai", "anthropic", "gemini", …).
	Provider string `json:"provider,omitempty"`

	// DisplayName is the human-readable name the provider advertises
	// (e.g. "Gemini 2.5 Flash"); empty when the provider lists only IDs.
	DisplayName string `json:"display_name,omitempty"`

	// Capabilities are the provider-advertised capabilities, e.g.
	// Gemini's supportedGenerationMethods ("generateContent",
	// "countTokens"). Nil when the provider does not advertise them.
	Capabilities []string `json:"capabilities,omitempty"`

	// ContextLimit is the provider-advertised input context window in
	// tokens; 0 when the provider does not advertise one.
	ContextLimit int `json:"context_limit,omitempty"`
}