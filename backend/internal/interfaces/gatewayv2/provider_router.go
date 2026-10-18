// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// ProviderRouterImpl routes requests to the appropriate upstream provider.
type ProviderRouterImpl struct {
	// Provider-specific configuration
	defaultBaseURL   string
	anthropicVersion string
	clock            func() time.Time
}

// NewProviderRouter creates a new provider router.
func NewProviderRouter(defaultBaseURL, anthropicVersion string) *ProviderRouterImpl {
	if defaultBaseURL == "" {
		defaultBaseURL = "https://api.anthropic.com"
	}
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}
	return &ProviderRouterImpl{
		defaultBaseURL:   defaultBaseURL,
		anthropicVersion: anthropicVersion,
		clock:            time.Now,
	}
}

// Route determines the upstream URL and headers for a request.
func (r *ProviderRouterImpl) Route(ctx context.Context, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	// Build upstream URL
	upstreamURL := upstream.BaseURL
	if upstreamURL == "" {
		upstreamURL = r.defaultBaseURL
	}

	// Ensure trailing slash for path joining
	upstreamURL = strings.TrimRight(upstreamURL, "/") + "/"

	// Create headers to inject
	headers := make(http.Header)
	headers.Set("x-api-key", upstream.UpstreamKey)
	headers.Set("anthropic-version", r.anthropicVersion)
	headers.Set("anthropic-beta", "output-128k-2025-02-19")

	// Add routing tags as headers for observability
	for k, v := range hints.RoutingTags {
		headers.Set("x-routing-"+k, v)
	}

	// Provider-specific handling based on provider kind
	switch upstream.ProviderAccount.ProviderKind {
	case "anthropic":
		return r.routeAnthropic(upstreamURL, headers, upstream, hints)
	case "openai":
		return r.routeOpenAI(upstreamURL, headers, upstream, hints)
	case "gemini":
		if upstream.BaseURL == "" {
			upstreamURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
		}
		return r.routeGemini(upstreamURL, headers, upstream, hints)
	case "azure_openai":
		return r.routeAzureOpenAI(upstreamURL, headers, upstream, hints)
	case "huggingface":
		if upstream.BaseURL == "" {
			upstreamURL = "https://api-inference.huggingface.co/v1/"
		}
		return r.routeHuggingFace(upstreamURL, headers, upstream, hints)
	case "custom":
		return r.routeCustom(upstreamURL, headers, upstream, hints)
	default:
		return upstreamURL, headers, nil
	}
}

// routeAnthropic handles Anthropic-specific routing.
func (r *ProviderRouterImpl) routeAnthropic(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	// Anthropic uses /v1/messages, /v1/messages/count_tokens, /v1/complete
	// The path is passed through from the original request
	return baseURL, headers, nil
}

// routeOpenAI handles OpenAI-compatible routing.
func (r *ProviderRouterImpl) routeOpenAI(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	// OpenAI-compatible providers use Authorization: Bearer
	if key := headers.Get("x-api-key"); key != "" {
		headers.Set("Authorization", "Bearer "+key)
		headers.Del("x-api-key")
	}
	headers.Del("anthropic-version")
	headers.Del("anthropic-beta")
	return baseURL, headers, nil
}

// routeHuggingFace handles Hugging Face provider routing with Authorization Bearer.
func (r *ProviderRouterImpl) routeHuggingFace(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	if key := headers.Get("x-api-key"); key != "" {
		headers.Set("Authorization", "Bearer "+key)
		headers.Del("x-api-key")
	}
	headers.Del("anthropic-version")
	headers.Del("anthropic-beta")
	return baseURL, headers, nil
}

// routeGemini handles Gemini-specific routing.
func (r *ProviderRouterImpl) routeGemini(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	headers.Set("Authorization", "Bearer "+upstream.UpstreamKey)
	headers.Set("x-goog-api-key", upstream.UpstreamKey)
	headers.Del("x-api-key")
	headers.Del("anthropic-version")
	headers.Del("anthropic-beta")
	return baseURL, headers, nil
}

// routeAzureOpenAI handles Azure OpenAI-specific routing.
func (r *ProviderRouterImpl) routeAzureOpenAI(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	headers.Set("api-key", upstream.UpstreamKey)
	headers.Del("x-api-key")
	headers.Del("anthropic-version")
	headers.Del("anthropic-beta")
	return baseURL, headers, nil
}

// routeCustom handles Custom provider routing with Authorization Bearer.
func (r *ProviderRouterImpl) routeCustom(baseURL string, headers http.Header, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error) {
	if key := headers.Get("x-api-key"); key != "" {
		headers.Set("Authorization", "Bearer "+key)
		headers.Del("x-api-key")
	}
	headers.Del("anthropic-version")
	headers.Del("anthropic-beta")
	return baseURL, headers, nil
}

// ProviderRouterImpl with request rewriting for provider compatibility.
type ProviderRewriter struct {
	router *ProviderRouterImpl
}

func NewProviderRewriter(router *ProviderRouterImpl) *ProviderRewriter {
	return &ProviderRewriter{router: router}
}

// RewriteForUpstream modifies the request for the target provider.
func (p *ProviderRewriter) RewriteForUpstream(r *http.Request, upstream ResolvedUpstream, hints InspectorHints) error {
	switch upstream.ProviderAccount.ProviderKind {
	case "anthropic":
		return p.rewriteForAnthropic(r, upstream, hints)
	case "openai", "huggingface":
		return p.rewriteForOpenAI(r, upstream, hints)
	}
	return nil
}

func (p *ProviderRewriter) rewriteForAnthropic(r *http.Request, upstream ResolvedUpstream, hints InspectorHints) error {
	// Anthropic-specific transformations
	// Ensure anthropic-version header
	if r.Header.Get("anthropic-version") == "" {
		r.Header.Set("anthropic-version", p.router.anthropicVersion)
	}
	// Pass through all other headers as-is
	return nil
}

func (p *ProviderRewriter) rewriteForOpenAI(r *http.Request, upstream ResolvedUpstream, hints InspectorHints) error {
	// OpenAI-compatible transformations
	// Convert x-api-key to Authorization: Bearer
	if key := r.Header.Get("x-api-key"); key != "" {
		r.Header.Set("Authorization", "Bearer "+key)
		r.Header.Del("x-api-key")
	}

	// Map Anthropic-specific headers to OpenAI equivalents
	// (if needed for compatibility layer)
	return nil
}

// UpstreamSelector provides intelligent upstream selection.
type UpstreamSelector struct {
	router *ProviderRouterImpl
}

func NewUpstreamSelector(router *ProviderRouterImpl) *UpstreamSelector {
	return &UpstreamSelector{router: router}
}

// SelectBestUpstream chooses the best upstream based on hints and health.
func (s *UpstreamSelector) SelectBestUpstream(ctx context.Context, upstreams []ResolvedUpstream, hints InspectorHints) (ResolvedUpstream, error) {
	if len(upstreams) == 0 {
		return ResolvedUpstream{}, ErrNoUpstream
	}
	if len(upstreams) == 1 {
		return upstreams[0], nil
	}

	// For now, return first healthy upstream
	// In production, implement health-aware routing, load balancing, etc.
	for _, u := range upstreams {
		// Could check health, latency, cost, etc.
		return u, nil
	}

	return upstreams[0], nil
}

type ProviderError struct {
	Type    string
	Message string
}

func (e *ProviderError) Error() string {
	return e.Type + ": " + e.Message
}

// ErrNoUpstream is the error returned when no healthy upstream is available.
var ErrNoUpstream = &ProviderError{Type: "upstream_error", Message: "no healthy upstream available"}
