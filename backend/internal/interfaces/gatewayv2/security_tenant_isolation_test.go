package gatewayv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

func TestGatewaySecurity_ScopeEscapeAndRestrictions(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.auth.cred.AllowedProviders = []string{"anthropic"}
	stubs.auth.cred.AllowedModels = []string{"claude-3-5-sonnet", "claude-sonnet-4-5"}

	t.Run("Credential cannot escape AllowedProviders to OpenAI", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "gpt-4o", Provider: "openai"}
		req := pipelineRequest("key-1", `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)
		res := g.executePipeline(context.Background(), req, "req-esc-openai")

		requireTrue(t, !res.Allowed, "openai model must be rejected for anthropic-bound credential")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})

	t.Run("Credential cannot escape AllowedProviders to Gemini", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-pro", Provider: "gemini"}
		req := pipelineRequest("key-1", `{"model":"gemini-1.5-pro","contents":[{"parts":[{"text":"hi"}]}]}`)
		res := g.executePipeline(context.Background(), req, "req-esc-gemini")

		requireTrue(t, !res.Allowed, "gemini model must be rejected for anthropic-bound credential")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})

	t.Run("Credential cannot escape AllowedProviders to Hugging Face", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "meta-llama/Llama-3.3-70B-Instruct", Provider: "huggingface"}
		req := pipelineRequest("key-1", `{"model":"meta-llama/Llama-3.3-70B-Instruct"}`)
		res := g.executePipeline(context.Background(), req, "req-esc-hf")

		requireTrue(t, !res.Allowed, "hugging face model must be rejected for anthropic-bound credential")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})

	t.Run("Credential cannot invoke unauthorized model even within allowed provider", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-3-opus-20240229", Provider: "anthropic"}
		req := pipelineRequest("key-1", `{"model":"claude-3-opus-20240229","messages":[{"role":"user","content":"hi"}]}`)
		res := g.executePipeline(context.Background(), req, "req-esc-opus")

		requireTrue(t, !res.Allowed, "unauthorized claude-3-opus must be rejected")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})

	t.Run("Credential with restricted models cannot omit model parameter to bypass scope", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "", Provider: "anthropic"}
		req := pipelineRequest("key-1", `{"messages":[{"role":"user","content":"hi"}]}`)
		res := g.executePipeline(context.Background(), req, "req-omit-model")

		requireTrue(t, !res.Allowed, "omitted model must be rejected for credential with allowed_models")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})
}

func TestGatewaySecurity_APIParameterTampering(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.auth.cred.AllowedProviders = []string{"anthropic"}
	stubs.auth.cred.AllowedModels = []string{"claude-3-5-sonnet"}

	t.Run("Conflicting query parameter model vs body model is rejected as parameter tampering", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?model=gpt-4o", strings.NewReader(`{"model":"claude-3-5-sonnet"}`))
		req.Header.Set("x-api-key", "key-1")
		req.Header.Set("Content-Type", "application/json")

		res := g.executePipeline(context.Background(), req, "req-tamper-query-model")
		requireTrue(t, !res.Allowed, "conflicting query model parameter must be rejected")
		requireEqual(t, res.StatusCode, http.StatusBadRequest)
		requireEqual(t, res.ErrorType, "parameter_tampering")
	})

	t.Run("Conflicting path model vs body model is rejected as parameter tampering", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-1.5-pro:generateContent", strings.NewReader(`{"model":"claude-3-5-sonnet"}`))
		req.Header.Set("x-api-key", "key-1")
		req.Header.Set("Content-Type", "application/json")

		res := g.executePipeline(context.Background(), req, "req-tamper-path-model")
		requireTrue(t, !res.Allowed, "conflicting path model parameter must be rejected")
		requireEqual(t, res.StatusCode, http.StatusBadRequest)
		requireEqual(t, res.ErrorType, "parameter_tampering")
	})

	t.Run("Conflicting query provider vs body provider is rejected as parameter tampering", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?provider=openai", strings.NewReader(`{"model":"claude-3-5-sonnet"}`))
		req.Header.Set("x-api-key", "key-1")
		req.Header.Set("Content-Type", "application/json")

		res := g.executePipeline(context.Background(), req, "req-tamper-query-provider")
		requireTrue(t, !res.Allowed, "conflicting query provider parameter must be rejected")
		requireEqual(t, res.StatusCode, http.StatusBadRequest)
		requireEqual(t, res.ErrorType, "parameter_tampering")
	})

	t.Run("Query model parameter is inspected and checked against AllowedModels when body model omitted", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "", Provider: "anthropic"}
		// Pass an unauthorized model via query parameter
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?model=gpt-4o", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("x-api-key", "key-1")
		req.Header.Set("Content-Type", "application/json")

		res := g.executePipeline(context.Background(), req, "req-tamper-query-only")
		requireTrue(t, !res.Allowed, "unauthorized query model must be rejected")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})
}

func TestGatewaySecurity_CrossTenantHeaderSpoofing(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	orgA := stubs.auth.cred.OrganizationID
	orgB := uuid.New()

	t.Run("Spoofed X-Organization-ID header is rejected with 403 tenant isolation violation", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		req.Header.Set("X-Organization-ID", orgB.String())

		res := g.executePipeline(context.Background(), req, "req-spoof-org")
		requireTrue(t, !res.Allowed, "spoofed org header must be rejected")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "tenant_isolation_violation")
	})

	t.Run("Spoofed X-User-ID header is rejected with 403 permission error", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
		stubs.auth.cred.SubjectType = "user"
		stubs.auth.cred.SubjectID = uuid.New()

		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		req.Header.Set("X-User-ID", uuid.New().String())

		res := g.executePipeline(context.Background(), req, "req-spoof-user")
		requireTrue(t, !res.Allowed, "spoofed user header must be rejected")
		requireEqual(t, res.StatusCode, http.StatusForbidden)
		requireEqual(t, res.ErrorType, "permission_error")
	})

	t.Run("Authentic matching headers are accepted", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
		stubs.auth.cred.AllowedProviders = nil
		stubs.auth.cred.AllowedModels = nil

		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		req.Header.Set("X-Organization-ID", orgA.String())

		res := g.executePipeline(context.Background(), req, "req-authentic-headers")
		requireTrue(t, res.Allowed, "matching authentic headers must be allowed: %s", res.ErrorMessage)
	})
}

func TestGatewaySecurity_RevocationAndDisablement(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	t.Run("Revoked gateway credential is immediately rejected with 401 authentication_error", func(t *testing.T) {
		stubs.auth.err = ErrCredentialRevoked
		req := pipelineRequest("key-revoked", `{"model":"claude-sonnet-4-5"}`)
		res := g.executePipeline(context.Background(), req, "req-revoked-key")

		requireTrue(t, !res.Allowed, "revoked credential must be rejected")
		requireEqual(t, res.StatusCode, http.StatusUnauthorized)
		requireEqual(t, res.ErrorType, "authentication_error")
	})

	t.Run("Disabled provider account is immediately rejected with 503 upstream_unavailable", func(t *testing.T) {
		stubs.auth.err = nil
		stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
		stubs.auth.cred.AllowedProviders = nil
		stubs.auth.cred.AllowedModels = nil

		disabledUpstream := upstreamFixture()
		disabledUpstream.ProviderAccount.Status = domain.ProviderStatusDisabled
		stubs.tenant.upstream = disabledUpstream

		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		res := g.executePipeline(context.Background(), req, "req-disabled-account")

		requireTrue(t, !res.Allowed, "disabled provider account must be rejected")
		requireEqual(t, res.StatusCode, http.StatusServiceUnavailable)
		requireEqual(t, res.ErrorType, "upstream_unavailable")
	})

	t.Run("Revoked provider credential is immediately rejected with 503 upstream_unavailable", func(t *testing.T) {
		stubs.auth.err = nil
		stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
		stubs.auth.cred.AllowedProviders = nil
		stubs.auth.cred.AllowedModels = nil

		revokedCredUpstream := upstreamFixture()
		revokedCredUpstream.Credential.Status = domain.ProviderCredentialStatusRevoked
		stubs.tenant.upstream = revokedCredUpstream

		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		res := g.executePipeline(context.Background(), req, "req-revoked-cred")

		requireTrue(t, !res.Allowed, "revoked provider credential must be rejected")
		requireEqual(t, res.StatusCode, http.StatusServiceUnavailable)
		requireEqual(t, res.ErrorType, "upstream_unavailable")
	})
}

func TestGatewaySecurity_SecretMaterialNonLeakage(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	secretUpstreamKey := "sk-ant-super-secret-upstream-token-xyz-12345"
	stubs.tenant.upstream.UpstreamKey = secretUpstreamKey
	stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}

	t.Run("Policy authorizer denial never exposes upstream key", func(t *testing.T) {
		stubs.policy.err = errors.New("policy store connection timed out")
		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		res := g.executePipeline(context.Background(), req, "req-secret-leak-policy")

		requireTrue(t, !res.Allowed, "denied request must not be allowed")
		requireTrue(t, !strings.Contains(res.ErrorMessage, secretUpstreamKey), "pipeline error must never expose upstream secret")
	})

	t.Run("Tenant resolver error never exposes upstream key", func(t *testing.T) {
		stubs.policy.err = nil
		stubs.tenant.err = errors.New("failed to load credentials for provider")
		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		res := g.executePipeline(context.Background(), req, "req-secret-leak-tenant")

		requireTrue(t, !res.Allowed, "tenant resolution failure must not be allowed")
		requireTrue(t, !strings.Contains(res.ErrorMessage, secretUpstreamKey), "tenant error must never expose upstream secret")
	})
}
