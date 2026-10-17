package gatewayv2

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGatewayLifecycle_StreamingSuccessAndTokenAttribution(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	firstToken := time.Now().UTC().Add(50 * time.Millisecond)
	stubs.proxy.usage = &StreamUsage{
		StatusCode:       http.StatusOK,
		Provider:         "anthropic",
		InputTokens:      150,
		OutputTokens:     300,
		CacheReadTokens:  50,
		CacheWriteTokens: 25,
		Reported:         true,
		FirstTokenAt:     firstToken,
	}

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, stubs.proxy.forwardCalled, "streamProxy.Forward must be called")
	requireEqual(t, len(stubs.telemetry.usage), 1)

	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.StatusCode, http.StatusOK)
	requireEqual(t, usage.InputTokens, 150)
	requireEqual(t, usage.OutputTokens, 300)
	requireEqual(t, usage.CacheReadTokens, 50)
	requireEqual(t, usage.CacheWriteTokens, 25)
	requireTrue(t, usage.TimeToFirstTokenMs != nil, "TTFT must be calculated")
	requireTrue(t, *usage.TimeToFirstTokenMs >= 0, "TTFT must be non-negative")
	requireEqual(t, usage.ProviderMeta["tokens_reported"], true)
}

func TestGatewayLifecycle_OpenAICompatibleReasoningTokens(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.proxy.usage = &StreamUsage{
		StatusCode:      http.StatusOK,
		Provider:        "openai",
		InputTokens:     200,
		OutputTokens:    400,
		ReasoningTokens: 120,
		Reported:        true,
	}

	body := `{"model": "o1-preview", "messages": [{"role": "user", "content": "think"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, stubs.proxy.forwardCalled, "streamProxy.Forward must be called for OpenAI route")
	requireEqual(t, len(stubs.telemetry.usage), 1)

	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.StatusCode, http.StatusOK)
	requireEqual(t, usage.InputTokens, 200)
	requireEqual(t, usage.OutputTokens, 400)
	requireEqual(t, usage.ProviderMeta["reasoning_tokens"], 120)
}

func TestGatewayLifecycle_UnreportedTokensExplicitNotFabricated(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Provider sends no token usage
	stubs.proxy.usage = nil

	body := `{"model": "claude-3-haiku-20240307", "messages": [{"role": "user", "content": "hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, stubs.proxy.forwardCalled, "streamProxy.Forward must be called")
	requireEqual(t, len(stubs.telemetry.usage), 1)

	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.ProviderMeta["tokens_reported"], false)
	requireEqual(t, usage.ProviderMeta["estimated"], true)
}

func TestGatewayLifecycle_UpstreamTimeoutMapping(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.proxy.err = ErrUpstreamTimeout

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "timeout"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusGatewayTimeout)
	requireTrue(t, strings.Contains(rec.Body.String(), "upstream_timeout"), "response body should contain upstream_timeout")
}

func TestGatewayLifecycle_UpstreamErrorMapping(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.proxy.err = ErrUpstreamError

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "err"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusBadGateway)
	requireTrue(t, strings.Contains(rec.Body.String(), "upstream_error"), "response body should contain upstream_error")
}

func TestGatewayLifecycle_ClientDisconnectHandledSafely(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.proxy.err = ErrClientDisconnected
	stubs.proxy.usage = &StreamUsage{
		StatusCode:   http.StatusOK,
		Provider:     "anthropic",
		InputTokens:  100,
		OutputTokens: 50,
		Reported:     true,
	}

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "disconnect"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Must not panic and must emit telemetry
	g.ServeHTTP(rec, req)

	requireEqual(t, len(stubs.telemetry.usage), 1)
	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.InputTokens, 100)
	requireEqual(t, usage.OutputTokens, 50)
}

func TestGatewayLifecycle_RevokedGatewayCredentialPreventsRequests(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.auth.err = ErrCredentialRevoked

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_revoked_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusUnauthorized)
	requireTrue(t, !stubs.proxy.forwardCalled, "revoked credential must never forward to upstream")
	requireTrue(t, strings.Contains(rec.Body.String(), "authentication_error"), "response should indicate authentication_error")
}

func TestGatewayLifecycle_ExpiredGatewayCredentialPreventsRequests(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.auth.err = ErrCredentialExpired

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_expired_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusUnauthorized)
	requireTrue(t, !stubs.proxy.forwardCalled, "expired credential must never forward to upstream")
}

func TestGatewayLifecycle_WrongScopeCredentialPreventsRequests(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.auth.err = ErrWrongScope

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_control_scope_only")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusForbidden)
	requireTrue(t, !stubs.proxy.forwardCalled, "wrong-scope credential must never forward to upstream")
	requireTrue(t, strings.Contains(rec.Body.String(), "permission_error"), "response should indicate permission_error")
}

func TestGatewayLifecycle_CrossTenantSpoofedHeaderRejected(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	// Spoofing header with a different tenant ID
	req.Header.Set("X-Organization-ID", "22222222-2222-2222-2222-222222222222")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusForbidden)
	requireTrue(t, !stubs.proxy.forwardCalled, "cross-tenant spoofing must never forward to upstream")
	requireTrue(t, strings.Contains(rec.Body.String(), "tenant_isolation_violation"), "response should indicate tenant_isolation_violation")
}

func TestGatewayLifecycle_GeminiRoutingAndUsage(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.inspect.hints.Model = "gemini-2.5-flash"
	stubs.tenant.upstream.ProviderAccount.ProviderKind = "gemini"

	firstToken := time.Now().UTC().Add(35 * time.Millisecond)
	stubs.proxy.usage = &StreamUsage{
		StatusCode:   http.StatusOK,
		Provider:     "gemini",
		InputTokens:  18,
		OutputTokens: 24,
		Reported:     true,
		FirstTokenAt: firstToken,
	}

	body := `{"model": "gemini-2.5-flash", "messages": [{"role": "user", "content": "hi gemini"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, stubs.proxy.forwardCalled, "streamProxy.Forward must be called for Gemini request")
	requireEqual(t, len(stubs.telemetry.usage), 1)

	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.StatusCode, http.StatusOK)
	requireEqual(t, usage.InputTokens, 18)
	requireEqual(t, usage.OutputTokens, 24)
	requireEqual(t, usage.ProviderMeta["tokens_reported"], true)
	requireEqual(t, usage.ProviderMeta["provider"], "gemini")
}

func TestGatewayLifecycle_UnsupportedModelRejected(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.inspect.hints.Model = "unsupported-hallucinated-model"

	body := `{"model": "unsupported-hallucinated-model", "messages": [{"role": "user", "content": "hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusBadRequest)
	requireTrue(t, !stubs.proxy.forwardCalled, "unsupported model must never forward to provider")
	requireTrue(t, strings.Contains(rec.Body.String(), "unsupported_model"), "response should indicate unsupported_model")
}

func TestGatewayLifecycle_RateLimitExceeded(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.rate.result = RateLimitResult{Allowed: false, Limit: 100, Remaining: 0, RetryAfter: time.Minute}

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "flood"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusTooManyRequests)
	requireTrue(t, !stubs.proxy.forwardCalled, "rate-limited request must never forward to provider")
	requireTrue(t, strings.Contains(rec.Body.String(), "rate_limit_exceeded"), "response should indicate rate_limit_exceeded")
}

func TestGatewayLifecycle_BudgetExceeded(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.budget.result = BudgetResult{Allowed: false, Limit: 100, Current: 100}

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "costly"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, rec.Code, http.StatusPaymentRequired)
	requireTrue(t, !stubs.proxy.forwardCalled, "budget-exceeded request must never forward to provider")
	requireTrue(t, strings.Contains(rec.Body.String(), "budget_exceeded"), "response should indicate budget_exceeded")
}

func TestGatewayLifecycle_InterruptedStreamTruncation(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	stubs.proxy.err = ErrStreamTruncated
	stubs.proxy.usage = &StreamUsage{
		StatusCode:   http.StatusOK,
		Provider:     "anthropic",
		InputTokens:  50,
		OutputTokens: 20,
		Reported:     true,
	}

	body := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "stream"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "aigw_valid_key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireEqual(t, len(stubs.telemetry.usage), 1)
	usage := stubs.telemetry.usage[0]
	requireEqual(t, usage.InputTokens, 50)
	requireEqual(t, usage.OutputTokens, 20)
}
