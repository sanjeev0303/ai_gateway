package gatewayv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCredentialScope_AnthropicBound_CannotInvokeOpenAIOrGemini(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Pinned / bound to anthropic only
	stubs.auth.cred.AllowedProviders = []string{"anthropic"}

	// 1. Attempt OpenAI model request -> rejected 403
	stubs.inspect.hints = InspectorHints{Model: "gpt-4o", Provider: "openai"}
	reqOpenAI := pipelineRequest("key-1", `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)
	resOpenAI := g.executePipeline(context.Background(), reqOpenAI, "req-openai-block")

	requireTrue(t, !resOpenAI.Allowed, "openai model must be rejected for anthropic-bound credential")
	requireEqual(t, resOpenAI.StatusCode, http.StatusForbidden)
	requireEqual(t, resOpenAI.ErrorType, "permission_error")

	// 2. Attempt Gemini model request -> rejected 403
	stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-pro", Provider: "gemini"}
	reqGemini := pipelineRequest("key-1", `{"model":"gemini-1.5-pro","contents":[{"parts":[{"text":"hi"}]}]}`)
	resGemini := g.executePipeline(context.Background(), reqGemini, "req-gemini-block")

	requireTrue(t, !resGemini.Allowed, "gemini model must be rejected for anthropic-bound credential")
	requireEqual(t, resGemini.StatusCode, http.StatusForbidden)
	requireEqual(t, resGemini.ErrorType, "permission_error")

	// 3. Permitted Anthropic model -> allowed 200
	stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
	stubs.tenant.upstream = upstreamFixture()
	reqAnthropic := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`)
	resAnthropic := g.executePipeline(context.Background(), reqAnthropic, "req-anthropic-ok")

	requireTrue(t, resAnthropic.Allowed, "anthropic model must be allowed: %s", resAnthropic.ErrorMessage)
}

func TestCredentialScope_GeminiBound_ModelRestrictions(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Bound to gemini provider and restricted to gemini-1.5-flash
	stubs.auth.cred.AllowedProviders = []string{"gemini"}
	stubs.auth.cred.AllowedModels = []string{"gemini-1.5-flash"}

	geminiUpstream := upstreamFixture()
	geminiUpstream.ProviderAccount.ProviderKind = "gemini"
	geminiUpstream.BaseURL = "https://generativelanguage.googleapis.com"
	stubs.tenant.upstream = geminiUpstream

	// 1. Attempt unauthorized Gemini model (gemini-1.5-pro) -> rejected 403
	stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-pro", Provider: "gemini"}
	reqPro := pipelineRequest("key-1", `{"model":"gemini-1.5-pro","contents":[{"parts":[{"text":"hi"}]}]}`)
	resPro := g.executePipeline(context.Background(), reqPro, "req-gemini-pro-block")

	requireTrue(t, !resPro.Allowed, "unauthorized gemini-1.5-pro must be rejected")
	requireEqual(t, resPro.StatusCode, http.StatusForbidden)
	requireEqual(t, resPro.ErrorType, "permission_error")

	// 2. Permitted Gemini model (gemini-1.5-flash) -> allowed
	stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-flash", Provider: "gemini"}
	reqFlash := pipelineRequest("key-1", `{"model":"gemini-1.5-flash","contents":[{"parts":[{"text":"hi"}]}]}`)
	resFlash := g.executePipeline(context.Background(), reqFlash, "req-gemini-flash-ok")

	requireTrue(t, resFlash.Allowed, "permitted gemini-1.5-flash must be allowed: %s", resFlash.ErrorMessage)
}

func TestCredentialScope_PinnedProviderAccount(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	pinnedAcctID := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	stubs.auth.cred.ProviderAccountID = &pinnedAcctID

	stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}
	req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`)
	res := g.executePipeline(context.Background(), req, "req-pinned-account")

	requireTrue(t, res.Allowed, "request must be allowed: %s", res.ErrorMessage)
	requireTrue(t, res.Context != nil, "request context must be populated")
	requireEqual(t, res.Context.Upstream.ProviderAccount.ID, pinnedAcctID, "must resolve to pinned provider account")
}

func TestCredentialScope_RevocationStopsRequestsImmediately(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Authenticator fails with revoked credential error
	stubs.auth.err = ErrCredentialRevoked

	req := pipelineRequest("key-revoked", `{"model":"claude-sonnet-4-5"}`)
	res := g.executePipeline(context.Background(), req, "req-revoked")

	requireTrue(t, !res.Allowed, "revoked credential must be rejected")
	requireEqual(t, res.StatusCode, http.StatusUnauthorized)
	requireEqual(t, res.ErrorType, "authentication_error")
}

func TestCredentialScope_ProviderRotationSafety(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Secret initial key
	secretKeyV1 := "sk-ant-api03-initial-secret-1111111111111111"
	stubs.tenant.upstream.UpstreamKey = secretKeyV1
	stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}

	req1 := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	res1 := g.executePipeline(context.Background(), req1, "req-rotate-1")
	requireTrue(t, res1.Allowed, "initial request must pass")
	requireEqual(t, res1.Context.Upstream.UpstreamKey, secretKeyV1)

	// Rotate upstream key
	secretKeyV2 := "sk-ant-api03-rotated-secret-2222222222222222"
	stubs.tenant.upstream.UpstreamKey = secretKeyV2

	req2 := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	res2 := g.executePipeline(context.Background(), req2, "req-rotate-2")
	requireTrue(t, res2.Allowed, "post-rotation request must pass")
	requireEqual(t, res2.Context.Upstream.UpstreamKey, secretKeyV2)

	// Ensure secret key is never leaked in error response
	stubs.auth.err = ErrWrongScope
	res3 := g.executePipeline(context.Background(), req2, "req-rotate-err")
	requireTrue(t, !res3.Allowed, "error request must be rejected")
	requireTrue(t, res3.ErrorMessage != secretKeyV1 && res3.ErrorMessage != secretKeyV2, "secret key must never leak in error messages")
}

func TestCredentialScope_HeaderAndParamSpoofingResistance(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Credential bound to anthropic
	stubs.auth.cred.AllowedProviders = []string{"anthropic"}
	stubs.inspect.hints = InspectorHints{Model: "claude-sonnet-4-5", Provider: "anthropic"}

	// 1. Caller attempts to override provider via x-provider header
	reqSpoof := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	reqSpoof.Header.Set("x-provider", "openai")

	resSpoof := g.executePipeline(context.Background(), reqSpoof, "req-spoof-header")
	requireTrue(t, !resSpoof.Allowed, "spoofed provider header must be rejected")
	requireEqual(t, resSpoof.StatusCode, http.StatusForbidden)
	requireEqual(t, resSpoof.ErrorType, "permission_error")

	// 2. Conflicting header even with open credential
	stubs.auth.cred.AllowedProviders = nil
	resConflict := g.executePipeline(context.Background(), reqSpoof, "req-conflict-header")
	requireTrue(t, !resConflict.Allowed, "conflicting provider header and model must be rejected")
	requireEqual(t, resConflict.StatusCode, http.StatusForbidden)
	requireEqual(t, resConflict.ErrorType, "permission_error")
}

func TestCredentialScope_ListModels_FilteredByScope(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	// Bound to gemini provider and restricted to gemini-1.5-flash
	stubs.auth.cred.AllowedProviders = []string{"gemini"}
	stubs.auth.cred.AllowedModels = []string{"gemini-1.5-flash"}

	// Set tenant resolver to report both anthropic and gemini as active
	stubTenant := &multiProviderStubTenant{
		providers: []string{"anthropic", "openai", "gemini"},
		upstreamMap: map[string]ResolvedUpstream{
			"gemini":    {ProviderAccount: ProviderAccount{ProviderKind: "gemini", Status: "active"}},
			"anthropic": {ProviderAccount: ProviderAccount{ProviderKind: "anthropic", Status: "active"}},
			"openai":    {ProviderAccount: ProviderAccount{ProviderKind: "openai", Status: "active"}},
		},
	}
	g.tenantResolver = stubTenant

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("x-api-key", "aigw_valid_key")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)
	requireEqual(t, rec.Code, http.StatusOK)

	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
		} `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	requireNoErr(t, err, "unmarshal models response")

	// Must contain exactly 1 model: gemini-1.5-flash
	requireEqual(t, len(resp.Data), 1)
	requireEqual(t, resp.Data[0].ID, "gemini-1.5-flash")
	requireEqual(t, resp.Data[0].Provider, "gemini")
}

func TestCredentialScope_DirectCredentialRateLimitAndBudget(t *testing.T) {
	rateLimit := 100
	budgetLimit := 50.0

	cred := GatewayCredential{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		RateLimit:      &rateLimit,
		BudgetLimit:    &budgetLimit,
	}

	snap := PolicySnapshot{
		OrganizationID: cred.OrganizationID,
	}

	// Verify policy authorizer returns applicable limits
	rates := snap.applicableRateLimits(cred)
	requireTrue(t, len(rates) >= 1, "expected at least 1 rate limit")
	requireEqual(t, rates[0].Limit, 100)
	requireEqual(t, rates[0].Window, time.Minute)

	budgets := snap.applicableBudgets(cred)
	requireTrue(t, len(budgets) >= 1, "expected at least 1 budget limit")
	requireEqual(t, budgets[0].Limit, 50.0)
	requireEqual(t, budgets[0].Window, 30*24*time.Hour)
}

type multiProviderStubTenant struct {
	providers   []string
	upstreamMap map[string]ResolvedUpstream
}

func (m *multiProviderStubTenant) ResolveUpstream(_ context.Context, _ uuid.UUID) (ResolvedUpstream, error) {
	if len(m.upstreamMap) > 0 {
		for _, up := range m.upstreamMap {
			return up, nil
		}
	}
	return upstreamFixture(), nil
}

func (m *multiProviderStubTenant) ResolveUpstreamForProvider(_ context.Context, _ uuid.UUID, providerKind string) (ResolvedUpstream, error) {
	if up, ok := m.upstreamMap[providerKind]; ok {
		return up, nil
	}
	up := upstreamFixture()
	up.ProviderAccount.ProviderKind = providerKind
	return up, nil
}

func (m *multiProviderStubTenant) ResolveUpstreamForAccount(_ context.Context, _ uuid.UUID, accountID uuid.UUID) (ResolvedUpstream, error) {
	up := upstreamFixture()
	up.ProviderAccount.ID = accountID
	return up, nil
}

func (m *multiProviderStubTenant) ListActiveProviders(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.providers, nil
}

func (m *multiProviderStubTenant) Invalidate(_ uuid.UUID) {}
