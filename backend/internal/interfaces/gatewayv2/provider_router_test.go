// Contract tests for the provider routing layer: URL construction,
// provider-specific header injection and upstream selection.
package gatewayv2

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRouter(t *testing.T) *ProviderRouterImpl {
	t.Helper()
	return NewProviderRouter("", "") // defaults: anthropic base URL + 2023-06-01
}

func TestProviderRouter_Route_Defaults(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "anthropic"},
		UpstreamKey:     "sk-ant-test",
	}

	url, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	if url != "https://api.anthropic.com/" {
		t.Fatalf("expected default base URL with trailing slash, got %q", url)
	}
	if got := headers.Get("x-api-key"); got != "sk-ant-test" {
		t.Fatalf("expected x-api-key header to be injected, got %q", got)
	}
	if got := headers.Get("anthropic-version"); got != "2023-06-01" {
		t.Fatalf("expected default anthropic-version, got %q", got)
	}
	if got := headers.Get("anthropic-beta"); got != "output-128k-2025-02-19" {
		t.Fatalf("expected anthropic-beta header, got %q", got)
	}
}

func TestProviderRouter_Route_CustomBaseURLAndVersion(t *testing.T) {
	r := NewProviderRouter("https://api.example.com", "2025-04-14")
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "anthropic"},
		UpstreamKey:     "sk-ant-test",
		BaseURL:         "https://api.other.com/v1", // account-level override wins
	}

	url, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	if url != "https://api.other.com/v1/" {
		t.Fatalf("expected account base URL with trailing slash, got %q", url)
	}
	if got := headers.Get("anthropic-version"); got != "2025-04-14" {
		t.Fatalf("expected custom anthropic-version, got %q", got)
	}
}

func TestProviderRouter_Route_RoutingTagsBecomeHeaders(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "anthropic"},
		UpstreamKey:     "sk-ant-test",
	}
	hints := InspectorHints{RoutingTags: map[string]string{"team": "eng", "env": "prod"}}

	_, headers, err := r.Route(context.Background(), upstream, hints)
	requireNoErr(t, err)

	if got := headers.Get("x-routing-team"); got != "eng" {
		t.Fatalf("expected x-routing-team header, got %q", got)
	}
	if got := headers.Get("x-routing-env"); got != "prod" {
		t.Fatalf("expected x-routing-env header, got %q", got)
	}
}

func TestProviderRouter_Route_AnthropicKeepsXAPIKey(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "anthropic"},
		UpstreamKey:     "sk-ant-test",
	}

	_, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	if headers.Get("x-api-key") == "" {
		t.Fatal("expected x-api-key to be kept for anthropic upstream")
	}
	if auth := headers.Get("Authorization"); auth != "" {
		t.Fatalf("expected no Authorization header for anthropic upstream, got %q", auth)
	}
}

func TestProviderRouter_Route_OpenAIConvertsToBearer(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "openai"},
		UpstreamKey:     "sk-proj-test",
	}

	_, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	if got := headers.Get("Authorization"); got != "Bearer sk-proj-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got)
	}
	if got := headers.Get("x-api-key"); got != "" {
		t.Fatalf("expected x-api-key to be removed for openai upstream, got %q", got)
	}
}

func TestProviderRouter_Route_UnknownKindPassthrough(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "cohere"},
		UpstreamKey:     "sk-coh-test",
	}

	_, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	// Unknown providers keep the generic header shape untouched.
	if headers.Get("x-api-key") != "sk-coh-test" {
		t.Fatalf("expected x-api-key passthrough, got %q", headers.Get("x-api-key"))
	}
	if headers.Get("Authorization") != "" {
		t.Fatalf("expected no Authorization header, got %q", headers.Get("Authorization"))
	}
}

func TestProviderRouter_Route_Gemini(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "gemini"},
		UpstreamKey:     "AIzaSyTestKey",
	}

	url, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)

	if headers.Get("x-goog-api-key") != "AIzaSyTestKey" {
		t.Fatalf("expected x-goog-api-key, got %q", headers.Get("x-goog-api-key"))
	}
	if headers.Get("x-api-key") != "" {
		t.Fatalf("expected no x-api-key header, got %q", headers.Get("x-api-key"))
	}
	if !strings.Contains(url, "googleapis.com") {
		t.Fatalf("expected googleapis.com in url, got %q", url)
	}
}

func TestProviderRewriter_AnthropicSetsDefaultVersionWhenMissing(t *testing.T) {
	p := NewProviderRewriter(testRouter(t))
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	upstream := ResolvedUpstream{ProviderAccount: ProviderAccount{ProviderKind: "anthropic"}}

	requireNoErr(t, p.RewriteForUpstream(req, upstream, InspectorHints{}))

	if got := req.Header.Get("anthropic-version"); got != "2023-06-01" {
		t.Fatalf("expected default anthropic-version injected, got %q", got)
	}
}

func TestProviderRewriter_AnthropicKeepsClientVersion(t *testing.T) {
	p := NewProviderRewriter(testRouter(t))
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("anthropic-version", "2025-01-01")
	upstream := ResolvedUpstream{ProviderAccount: ProviderAccount{ProviderKind: "anthropic"}}

	requireNoErr(t, p.RewriteForUpstream(req, upstream, InspectorHints{}))

	if got := req.Header.Get("anthropic-version"); got != "2025-01-01" {
		t.Fatalf("expected client-provided version preserved, got %q", got)
	}
}

func TestProviderRewriter_OpenAIConvertsHeader(t *testing.T) {
	p := NewProviderRewriter(testRouter(t))
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("x-api-key", "sk-proj-test")
	upstream := ResolvedUpstream{ProviderAccount: ProviderAccount{ProviderKind: "openai"}}

	requireNoErr(t, p.RewriteForUpstream(req, upstream, InspectorHints{}))

	if got := req.Header.Get("Authorization"); got != "Bearer sk-proj-test" {
		t.Fatalf("expected Authorization Bearer, got %q", got)
	}
	if got := req.Header.Get("x-api-key"); got != "" {
		t.Fatalf("expected x-api-key removed, got %q", got)
	}
}

func TestProviderRewriter_UnknownKindNoOp(t *testing.T) {
	p := NewProviderRewriter(testRouter(t))
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-gem-test")
	upstream := ResolvedUpstream{ProviderAccount: ProviderAccount{ProviderKind: "gemini"}}

	requireNoErr(t, p.RewriteForUpstream(req, upstream, InspectorHints{}))

	if req.Header.Get("x-api-key") != "sk-gem-test" {
		t.Fatal("expected request untouched for unknown provider kind")
	}
}

func TestUpstreamSelector_EmptyReturnsErrNoUpstream(t *testing.T) {
	s := NewUpstreamSelector(testRouter(t))

	_, err := s.SelectBestUpstream(context.Background(), nil, InspectorHints{})

	if !errors.Is(err, ErrNoUpstream) {
		t.Fatalf("expected ErrNoUpstream, got %v", err)
	}
}

func TestUpstreamSelector_SingleAndMultiplePickFirst(t *testing.T) {
	s := NewUpstreamSelector(testRouter(t))
	first := ResolvedUpstream{ProviderAccount: ProviderAccount{ID: mustUUIDs[0], ProviderKind: "anthropic"}}
	second := ResolvedUpstream{ProviderAccount: ProviderAccount{ID: mustUUIDs[1], ProviderKind: "openai"}}

	for _, upstreams := range [][]ResolvedUpstream{{first}, {first, second}} {
		got, err := s.SelectBestUpstream(context.Background(), upstreams, InspectorHints{})
		requireNoErr(t, err)
		if got.ProviderAccount.ID != first.ProviderAccount.ID {
			t.Fatalf("expected first upstream selected, got %v", got.ProviderAccount.ID)
		}
	}
}

func TestProviderError_ErrorString(t *testing.T) {
	err := &ProviderError{Type: "upstream_error", Message: "no healthy upstream available"}
	if !strings.Contains(err.Error(), "upstream_error") || !strings.Contains(err.Error(), "no healthy upstream available") {
		t.Fatalf("expected type and message in error string, got %q", err.Error())
	}
}

func TestProviderError_IsErrNoUpstream(t *testing.T) {
	if !errors.Is(ErrNoUpstream, ErrNoUpstream) {
		t.Fatal("ErrNoUpstream must wrap itself")
	}
	if errors.Is(&ProviderError{Type: "other", Message: "x"}, ErrNoUpstream) {
		t.Fatal("unrelated ProviderError must not match ErrNoUpstream")
	}
}

func TestValidateHFEndpoint_SSRFProtection(t *testing.T) {
	valid := []string{
		"https://my-custom-endpoint.us-east-1.aws.endpoints.huggingface.cloud",
		"https://router.huggingface.co/v1",
		"https://api-inference.huggingface.co/v1",
		"https://custom.huggingface.co",
	}
	for _, ep := range valid {
		if err := ValidateHFEndpoint(ep); err != nil {
			t.Fatalf("expected valid HF endpoint for %q, got error: %v", ep, err)
		}
	}

	invalid := []string{
		"http://my-custom-endpoint.endpoints.huggingface.cloud", // HTTP scheme rejected
		"http://169.254.169.254/latest/meta-data",               // Cloud metadata
		"https://127.0.0.1:8080/admin",                          // Loopback
		"https://10.0.0.1/private",                              // Private RFC1918
		"https://192.168.1.1/secret",                            // Private RFC1918
		"https://metadata.google.internal/compute",              // GCP metadata
		"https://localhost:443",                                 // Localhost
		"https://attacker.com/steal",                            // Non-HF domain
	}
	for _, ep := range invalid {
		if err := ValidateHFEndpoint(ep); err == nil {
			t.Fatalf("expected SSRF error for %q, got nil", ep)
		}
	}
}

func TestProviderRouter_Route_HuggingFaceDedicatedEndpoint(t *testing.T) {
	r := testRouter(t)
	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{ProviderKind: "huggingface"},
		UpstreamKey:     "hf_token_123",
	}

	// 1. Default Hub endpoint
	url, headers, err := r.Route(context.Background(), upstream, InspectorHints{})
	requireNoErr(t, err)
	if url != "https://api-inference.huggingface.co/v1/" {
		t.Fatalf("expected default HF hub router URL, got %q", url)
	}
	if got := headers.Get("Authorization"); got != "Bearer hf_token_123" {
		t.Fatalf("expected Authorization Bearer header, got %q", got)
	}

	// 2. Verified dedicated HF inference endpoint
	customEP := "https://my-endpoint.us-east-1.aws.endpoints.huggingface.cloud"
	url, _, err = r.Route(context.Background(), upstream, InspectorHints{CustomEndpoint: customEP})
	requireNoErr(t, err)
	if url != "https://my-endpoint.us-east-1.aws.endpoints.huggingface.cloud/" {
		t.Fatalf("expected custom endpoint URL with trailing slash, got %q", url)
	}

	// 3. Malicious SSRF endpoint rejected
	badEP := "https://169.254.169.254/latest/meta-data"
	_, _, err = r.Route(context.Background(), upstream, InspectorHints{CustomEndpoint: badEP})
	if err == nil {
		t.Fatal("expected SSRF validation error, got nil")
	}
}
