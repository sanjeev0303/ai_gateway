// Package gatewayv2 contract tests for the gateway pipeline: every stage of
// executePipeline, HTTP entry point, telemetry emission and lifecycle methods.
package gatewayv2

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ---- Pipeline component stubs -------------------------------------------------

type stubGatewayAuthenticator struct {
	cred GatewayCredential
	err  error
}

var _ Authenticator = (*stubGatewayAuthenticator)(nil)

func (s *stubGatewayAuthenticator) Authenticate(_ context.Context, _ string) (GatewayCredential, error) {
	return s.cred, s.err
}

type stubGatewayTenantResolver struct {
	upstream ResolvedUpstream
	err      error
}

var _ TenantResolver = (*stubGatewayTenantResolver)(nil)

func (s *stubGatewayTenantResolver) ResolveUpstream(_ context.Context, _ uuid.UUID) (ResolvedUpstream, error) {
	return s.upstream, s.err
}

type stubGatewayPolicyResolver struct {
	snapshot         PolicySnapshot
	err              error
	invalidatedOrg   uuid.UUID
	invalidateCalled bool
}

var _ PolicyResolver = (*stubGatewayPolicyResolver)(nil)

func (s *stubGatewayPolicyResolver) GetSnapshot(_ context.Context, _ uuid.UUID) (PolicySnapshot, error) {
	return s.snapshot, s.err
}
func (s *stubGatewayPolicyResolver) Invalidate(orgID uuid.UUID) {
	s.invalidatedOrg = orgID
	s.invalidateCalled = true
}

type stubGatewayRateLimiter struct {
	result     RateLimitResult
	lastKey    string
	lastLimit  int
	lastWindow time.Duration
}

var _ RateLimiter = (*stubGatewayRateLimiter)(nil)

func (s *stubGatewayRateLimiter) Check(_ context.Context, key string, limit int, window time.Duration) RateLimitResult {
	s.lastKey = key
	s.lastLimit = limit
	s.lastWindow = window
	return s.result
}

type stubGatewayBudgetGuard struct {
	result     BudgetResult
	lastKey    string
	lastLimit  float64
	lastWindow time.Duration
	lastCost   float64
}

var _ BudgetGuard = (*stubGatewayBudgetGuard)(nil)

func (s *stubGatewayBudgetGuard) Check(_ context.Context, key string, limit float64, window time.Duration, cost float64) BudgetResult {
	s.lastKey = key
	s.lastLimit = limit
	s.lastWindow = window
	s.lastCost = cost
	return s.result
}
func (s *stubGatewayBudgetGuard) RecordUsage(_ context.Context, _ string, _ float64) error {
	return nil
}

type stubGatewayRequestInspector struct {
	hints InspectorHints
	err   error
}

var _ RequestInspector = (*stubGatewayRequestInspector)(nil)

func (s *stubGatewayRequestInspector) Inspect(_ context.Context, _ io.Reader) (InspectorHints, error) {
	return s.hints, s.err
}

type stubGatewayProviderRouter struct {
	url     string
	headers http.Header
	err     error
}

var _ ProviderRouter = (*stubGatewayProviderRouter)(nil)

func (s *stubGatewayProviderRouter) Route(_ context.Context, _ ResolvedUpstream, _ InspectorHints) (string, http.Header, error) {
	return s.url, s.headers, s.err
}

type stubGatewayStreamProxy struct {
	forwardCalled bool
	closeCalled   bool
	usage         *StreamUsage
	err           error
}

var _ StreamProxy = (*stubGatewayStreamProxy)(nil)

func (s *stubGatewayStreamProxy) Forward(context.Context, http.ResponseWriter, *http.Request, string, http.Header) (*StreamUsage, error) {
	s.forwardCalled = true
	return s.usage, s.err
}
func (s *stubGatewayStreamProxy) Close(time.Duration) error {
	s.closeCalled = true
	return nil
}

type stubGatewayTelemetry struct {
	usage       []UsageRecord
	errs        []ErrorRecord
	flushCalled bool
	closeCalled bool
}

var _ TelemetryEmitter = (*stubGatewayTelemetry)(nil)

func (s *stubGatewayTelemetry) EmitUsage(_ context.Context, u UsageRecord) {
	s.usage = append(s.usage, u)
}
func (s *stubGatewayTelemetry) EmitError(_ context.Context, e ErrorRecord) {
	s.errs = append(s.errs, e)
}
func (s *stubGatewayTelemetry) Flush(context.Context) error {
	s.flushCalled = true
	return nil
}
func (s *stubGatewayTelemetry) Close() error {
	s.closeCalled = true
	return nil
}

// gatewayStubs bundles the pipeline stubs for a test gateway.
type gatewayStubs struct {
	auth      *stubGatewayAuthenticator
	tenant    *stubGatewayTenantResolver
	policy    *stubGatewayPolicyResolver
	rate      *stubGatewayRateLimiter
	budget    *stubGatewayBudgetGuard
	inspect   *stubGatewayRequestInspector
	router    *stubGatewayProviderRouter
	proxy     *stubGatewayStreamProxy
	telemetry *stubGatewayTelemetry
}

// newGatewayForPipeline builds a Gateway wired to stubs where every stage
// succeeds by default; tests flip one stub to exercise a failure path.
func newGatewayForPipeline() (*Gateway, *gatewayStubs) {
	cred := gatewayCredFixture()
	stubs := &gatewayStubs{
		auth:   &stubGatewayAuthenticator{cred: cred},
		tenant: &stubGatewayTenantResolver{upstream: upstreamFixture()},
		policy: &stubGatewayPolicyResolver{snapshot: PolicySnapshot{OrganizationID: cred.OrganizationID}},
		rate:   &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true, Limit: 1000, Remaining: 999}},
		budget: &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true, Limit: 1000}},
		inspect: &stubGatewayRequestInspector{hints: InspectorHints{
			Model:                 "claude-sonnet-4-5",
			EstimatedInputTokens:  1000,
			EstimatedOutputTokens: 2000,
		}},
		router: &stubGatewayProviderRouter{
			url:     "https://api.test/v1/messages",
			headers: http.Header{"X-Upstream-Key": {"k"}},
		},
		proxy:     &stubGatewayStreamProxy{},
		telemetry: &stubGatewayTelemetry{},
	}
	g := &Gateway{
		config:           DefaultGatewayConfig(),
		logger:           discardLogger(),
		authenticator:    stubs.auth,
		tenantResolver:   stubs.tenant,
		policyResolver:   stubs.policy,
		rateLimiter:      stubs.rate,
		budgetGuard:      stubs.budget,
		requestInspector: stubs.inspect,
		providerRouter:   stubs.router,
		streamProxy:      stubs.proxy,
		telemetry:        stubs.telemetry,
	}
	return g, stubs
}

func gatewayCredFixture() GatewayCredential {
	return GatewayCredential{
		ID:             mustUUIDs[0],
		OrganizationID: mustUUIDs[1],
		SubjectType:    "service_account",
		SubjectID:      mustUUIDs[2],
		Scopes:         []string{"gateway"},
		Status:         "active",
	}
}

func upstreamFixture() ResolvedUpstream {
	return ResolvedUpstream{
		ProviderAccount: ProviderAccount{
			ID:             mustUUIDs[0],
			OrganizationID: mustUUIDs[1],
			ProviderKind:   "anthropic",
			Name:           "main",
			BaseURL:        "https://api.anthropic.com",
			Status:         "active",
		},
		Credential: ProviderCredential{
			ID:                mustUUIDs[2],
			OrganizationID:    mustUUIDs[1],
			ProviderAccountID: mustUUIDs[0],
			SecretRef:         "ref:main",
			Status:            "active",
		},
		UpstreamKey: "sk-test-1234",
		BaseURL:     "https://api.anthropic.com",
	}
}

// pipelineRequest builds a POST request with an optional x-api-key header.
func pipelineRequest(apiKey, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	return req
}

// ---- NewGateway wiring -------------------------------------------------------

// TestGateway_NewGatewayWiring verifies zero config gets defaults and every
// component is constructed from the dependency set.
func TestGateway_NewGatewayWiring(t *testing.T) {
	g, err := NewGateway(GatewayDeps{
		Config:              GatewayConfig{},
		Logger:              discardLogger(),
		Hasher:              &stubHasher{},
		GatewayCredRepo:     &stubGatewayCredRepo{},
		ProviderAccounts:    &stubAccountRepo{},
		ProviderCredentials: &stubCredentialRepo{},
		Secrets:             &stubSecretStore{},
		PolicyRepo:          &stubPolicyRepo{},
		RuleRepo:            &stubPolicyRuleRepo{},
	})
	requireNoErr(t, err, "NewGateway")

	requireTrue(t, g.config == DefaultGatewayConfig(), "zero config should fall back to defaults")

	requireTrue(t, g.authenticator != nil, "authenticator not wired")
	requireTrue(t, g.tenantResolver != nil, "tenantResolver not wired")
	requireTrue(t, g.policyResolver != nil, "policyResolver not wired")
	requireTrue(t, g.rateLimiter != nil, "rateLimiter not wired")
	requireTrue(t, g.budgetGuard != nil, "budgetGuard not wired")
	requireTrue(t, g.requestInspector != nil, "requestInspector not wired")
	requireTrue(t, g.providerRouter != nil, "providerRouter not wired")
	requireTrue(t, g.streamProxy != nil, "streamProxy not wired")
	requireTrue(t, g.telemetry != nil, "telemetry not wired")

	t.Cleanup(func() {
		requireNoErr(t, g.Shutdown(0), "Shutdown")
		// Shutdown does not stop the rate limiter's cleanup goroutine.
		g.rateLimiter.(*RateLimiterImpl).Close()
	})
}

// TestGateway_NewGatewayOverrides verifies optional dependency overrides win
// over the default constructions.
func TestGateway_NewGatewayOverrides(t *testing.T) {
	custom := &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true}}
	customBudget := &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true}}
	customTelemetry := &stubGatewayTelemetry{}

	g, err := NewGateway(GatewayDeps{
		Config:              DefaultGatewayConfig(),
		Hasher:              &stubHasher{},
		GatewayCredRepo:     &stubGatewayCredRepo{},
		ProviderAccounts:    &stubAccountRepo{},
		ProviderCredentials: &stubCredentialRepo{},
		Secrets:             &stubSecretStore{},
		PolicyRepo:          &stubPolicyRepo{},
		RuleRepo:            &stubPolicyRuleRepo{},
		RateLimiter:         custom,
		BudgetGuard:         customBudget,
		Telemetry:           customTelemetry,
	})
	requireNoErr(t, err, "NewGateway")

	requireTrue(t, g.rateLimiter == custom, "custom rateLimiter not honored")
	requireTrue(t, g.budgetGuard == customBudget, "custom budgetGuard not honored")
	requireTrue(t, g.telemetry == customTelemetry, "custom telemetry not honored")
}

// ---- executePipeline stages --------------------------------------------------

// TestExecutePipeline_RejectsMissingAPIKey covers stage 1 (authenticate):
// no key at all must fail before the authenticator is consulted.
func TestExecutePipeline_RejectsMissingAPIKey(t *testing.T) {
	g, _ := newGatewayForPipeline()
	res := g.executePipeline(context.Background(), pipelineRequest("", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "request without key must be denied")
	requireTrue(t, res.StatusCode == http.StatusUnauthorized, "want 401, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "authentication_error", "want authentication_error, got %s", res.ErrorType)
}

// TestExecutePipeline_AuthenticatorError covers stage 1: authenticator
// failures map to 401 authentication_error.
func TestExecutePipeline_AuthenticatorError(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.auth.err = errStub
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "auth error must be denied")
	requireTrue(t, res.StatusCode == http.StatusUnauthorized, "want 401, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "authentication_error", "want authentication_error, got %s", res.ErrorType)
	requireTrue(t, res.ErrorMessage == errStub.Error(), "error message not propagated")
}

// TestExecutePipeline_TenantResolverError covers stage 2: resoluFailure maps
// to 503 upstream_unavailable.
func TestExecutePipeline_TenantResolverError(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.tenant.err = errStub
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "tenant error must be denied")
	requireTrue(t, res.StatusCode == http.StatusServiceUnavailable, "want 503, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "upstream_unavailable", "want upstream_unavailable, got %s", res.ErrorType)
}

// TestExecutePipeline_PolicyErrorFailsClosed covers stage 3: an unavailable
// policy snapshot denies the request with 503 policy_unavailable.
func TestExecutePipeline_PolicyErrorFailsClosed(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.err = errStub

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "policy failure must deny the request")
	requireTrue(t, res.StatusCode == http.StatusServiceUnavailable, "want 503, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "policy_unavailable", "want policy_unavailable, got %s", res.ErrorType)
}

// TestExecutePipeline_ModelAllowlist covers stage 5: an allowlist rule admits
// the listed model and denies every other model.
func TestExecutePipeline_ModelAllowlist(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.snapshot.Rules = []PolicyRule{modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5")}
	stubs.inspect.hints.Model = "claude-sonnet-4-5"

	allowed := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	requireTrue(t, allowed.Allowed, "listed model must be allowed")

	stubs.inspect.hints.Model = "gpt-4o"
	denied := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-2")
	requireTrue(t, !denied.Allowed, "unlisted model must be denied")
	requireTrue(t, denied.StatusCode == http.StatusForbidden, "want 403, got %d", denied.StatusCode)
	requireTrue(t, denied.ErrorType == "policy_denied", "want policy_denied, got %s", denied.ErrorType)
}

// TestExecutePipeline_ModelBlocklist covers stage 5: a blocklist rule denies
// the listed model and admits every other model.
func TestExecutePipeline_ModelBlocklist(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.snapshot.Rules = []PolicyRule{modelRule(domain.RuleTypeBlocklist, "gpt-4o")}
	stubs.inspect.hints.Model = "claude-sonnet-4-5"

	allowed := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	requireTrue(t, allowed.Allowed, "non-blocked model must be allowed")

	stubs.inspect.hints.Model = "gpt-4o"
	denied := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-2")
	requireTrue(t, !denied.Allowed, "blocked model must be denied")
	requireTrue(t, denied.StatusCode == http.StatusForbidden, "want 403, got %d", denied.StatusCode)
	requireTrue(t, denied.ErrorType == "policy_denied", "want policy_denied, got %s", denied.ErrorType)
}

// TestExecutePipeline_EmptyModelSkipsAuthorization covers stage 5: an unknown
// model (inspector gap) is not blocked by allowlists, so one failed inspection
// cannot take down an org's traffic.
func TestExecutePipeline_EmptyModelSkipsAuthorization(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.snapshot.Rules = []PolicyRule{modelRule(domain.RuleTypeAllowlist, "claude-sonnet-4-5")}
	stubs.inspect.hints.Model = ""

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	requireTrue(t, res.Allowed, "unknown model must not be denied by an allowlist")
}

// TestExecutePipeline_SnapshotRateLimitUsed covers stage 6: a credential-scoped
// snapshot rate config overrides the gateway defaults.
func TestExecutePipeline_SnapshotRateLimitUsed(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.snapshot.RateLimits = map[string]RateLimitConfig{
		"cred:infer": {Limit: 5, Window: 30 * time.Second, Scope: "credential"},
	}

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	requireTrue(t, res.Allowed, "request must pass with the snapshot rate config")
	requireTrue(t, stubs.rate.lastLimit == 5, "snapshot rate limit not honored: %d", stubs.rate.lastLimit)
	requireTrue(t, stubs.rate.lastWindow == 30*time.Second, "snapshot rate window not honored: %v", stubs.rate.lastWindow)
}

// TestExecutePipeline_SnapshotBudgetUsed covers stage 7: an org-scoped snapshot
// budget config overrides the gateway defaults.
func TestExecutePipeline_SnapshotBudgetUsed(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.policy.snapshot.Budgets = map[string]BudgetConfig{
		"org:budget": {Limit: 42, Window: 45 * time.Minute, Scope: "organization"},
	}

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	requireTrue(t, res.Allowed, "request must pass with the snapshot budget config")
	requireTrue(t, stubs.budget.lastLimit == 42, "snapshot budget limit not honored: %v", stubs.budget.lastLimit)
	requireTrue(t, stubs.budget.lastWindow == 45*time.Minute, "snapshot budget window not honored: %v", stubs.budget.lastWindow)
}

// TestExecutePipeline_RateLimited covers stage 4: a denied rate check maps to
// 429 rate_limit_exceeded carrying the full rate limit result.
func TestExecutePipeline_RateLimited(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.rate.result = RateLimitResult{
		Allowed:    false,
		Limit:      1000,
		Remaining:  0,
		RetryAfter: time.Minute,
		Scope:      "credential",
		Key:        "cred:" + gatewayCredFixture().ID.String(),
	}
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "rate limited request must be denied")
	requireTrue(t, res.StatusCode == 429, "want 429, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "rate_limit_exceeded", "want rate_limit_exceeded, got %s", res.ErrorType)
	requireTrue(t, res.Context != nil && !res.Context.RateLimit.Allowed, "RateLimit result not preserved")
}

// TestExecutePipeline_InspectErrorFallsBack covers stage 5: inspection failure
// warns and continues with empty hints.
func TestExecutePipeline_InspectErrorFallsBack(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.inspect.err = errStub
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, res.Allowed, "inspection failure must not block the request")
	requireTrue(t, res.Context.InspectorHints.Model == "", "expected empty hints on inspection failure")

	// Routing still happens with empty hints.
	requireTrue(t, res.Context.UpstreamURL == "https://api.test/v1/messages", "router not reached")
}

// TestExecutePipeline_BudgetExceeded covers stage 6: a denied budget check maps
// to 402 budget_exceeded carrying the budget result, and the budget checker
// receives the org-scoped key plus the estimated cost.
func TestExecutePipeline_BudgetExceeded(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.budget.result = BudgetResult{Allowed: false, Limit: 1000}
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "budget exceeded request must be denied")
	requireTrue(t, res.StatusCode == 402, "want 402, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "budget_exceeded", "want budget_exceeded, got %s", res.ErrorType)
	requireTrue(t, res.Context != nil && !res.Context.BudgetResult.Allowed, "BudgetResult not preserved")

	cred := gatewayCredFixture()
	requireTrue(t, stubs.budget.lastKey == "org:"+cred.OrganizationID.String(), "budget key mismatch: %s", stubs.budget.lastKey)
	requireTrue(t, stubs.budget.lastLimit == DefaultGatewayConfig().DefaultBudgetLimit, "budget limit mismatch")
	requireTrue(t, stubs.budget.lastWindow == DefaultGatewayConfig().DefaultBudgetWindow, "budget window mismatch")

	wantCost := estimateCostForFixture()
	if math.Abs(stubs.budget.lastCost-wantCost) > 1e-9 {
		t.Fatalf("estimated cost mismatch: got %v, want %v", stubs.budget.lastCost, wantCost)
	}
}

// TestExecutePipeline_RouteError covers stage 7: routing failure maps to
// 500 routing_error.
func TestExecutePipeline_RouteError(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.router.err = errStub
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, !res.Allowed, "route error must be denied")
	requireTrue(t, res.StatusCode == http.StatusInternalServerError, "want 500, got %d", res.StatusCode)
	requireTrue(t, res.ErrorType == "routing_error", "want routing_error, got %s", res.ErrorType)
}

// TestExecutePipeline_AllowedPopulatesContext covers the happy path: every
// stage result lands on the request context and the rate limiter receives the
// credential-scoped key with the configured defaults.
func TestExecutePipeline_AllowedPopulatesContext(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	cred := gatewayCredFixture()
	hints := InspectorHints{Model: "claude-sonnet-4-5", EstimatedInputTokens: 1000, EstimatedOutputTokens: 2000}

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	requireTrue(t, res.Allowed, "happy path must be allowed")
	requireTrue(t, res.StatusCode == http.StatusOK, "want 200, got %d", res.StatusCode)
	requireTrue(t, res.Context != nil, "context missing on allowed result")

	ctx := res.Context
	requireTrue(t, ctx.Credential.ID == cred.ID, "credential not preserved")
	requireTrue(t, ctx.Upstream.BaseURL == "https://api.anthropic.com", "upstream not preserved")
	requireTrue(t, ctx.PolicySnapshot.OrganizationID == cred.OrganizationID, "policy snapshot not preserved")
	requireTrue(t, ctx.RateLimit.Allowed, "rate limit result not preserved")
	requireTrue(t, ctx.BudgetResult.Allowed, "budget result not preserved")
	requireTrue(t, ctx.InspectorHints.Model == hints.Model, "hints not preserved")
	requireTrue(t, !ctx.StartTime.IsZero(), "start time not set")
	requireTrue(t, ctx.UpstreamURL == "https://api.test/v1/messages", "upstream url mismatch")
	requireTrue(t, ctx.UpstreamHeaders.Get("x-upstream-key") == "k", "upstream headers mismatch")

	requireTrue(t, stubs.rate.lastKey == "cred:"+cred.ID.String(), "rate key mismatch: %s", stubs.rate.lastKey)
	requireTrue(t, stubs.rate.lastLimit == DefaultGatewayConfig().DefaultRateLimit, "rate limit mismatch")
	requireTrue(t, stubs.rate.lastWindow == DefaultGatewayConfig().DefaultRateWindow, "rate window mismatch")
}

// ---- HTTP entry point --------------------------------------------------------

// TestServeHTTP_RejectsNonPost verifies only POST is served and the response
// carries an Anthropic-compatible error envelope.
func TestServeHTTP_RejectsNonPost(t *testing.T) {
	g, _ := newGatewayForPipeline()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, rec.Code == http.StatusMethodNotAllowed, "want 405, got %d", rec.Code)
	requireTrue(t, rec.Header().Get("x-request-id") != "", "x-request-id not set")

	var body anthropicErrorBody
	requireNoErr(t, json.NewDecoder(rec.Body).Decode(&body), "decode error body")
	requireTrue(t, body.Type == "error", "want error envelope, got %s", body.Type)
	requireTrue(t, body.Error.Type == "method_not_allowed", "want method_not_allowed, got %s", body.Error.Type)
}

// TestServeHTTP_UnauthorizedWritesErrorJSON verifies pipeline rejection status
// and envelope reach the client with the request id header.
func TestServeHTTP_UnauthorizedWritesErrorJSON(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.auth.err = errStub
	req := pipelineRequest("key-1", "{}")
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, rec.Code == http.StatusUnauthorized, "want 401, got %d", rec.Code)
	requireTrue(t, rec.Header().Get("x-request-id") != "", "x-request-id not set")

	var body anthropicErrorBody
	requireNoErr(t, json.NewDecoder(rec.Body).Decode(&body), "decode error body")
	requireTrue(t, body.Error.Type == "authentication_error", "want authentication_error, got %s", body.Error.Type)
	requireTrue(t, body.Error.Message == errStub.Error(), "error message mismatch")
}

// TestServeHTTP_AllowedForwardsUpstream verifies the allowed path actually hands
// the request to the stream proxy using the routed upstream. Before the fix the
// gateway wrote a bare 200 and never called Forward at all.
func TestServeHTTP_AllowedForwardsUpstream(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	requireTrue(t, stubs.proxy.forwardCalled, "allowed request was never forwarded upstream")
	requireTrue(t, rec.Header().Get("x-request-id") != "", "x-request-id not set")
}

// ---- extractAPIKey -----------------------------------------------------------

// TestExtractAPIKeyPinpoints the header precedence and normalization rules.
func TestExtractAPIKey(t *testing.T) {
	g, _ := newGatewayForPipeline()

	// x-api-key wins over Authorization: Bearer.
	req := pipelineRequest("key-api", "{}")
	req.Header.Set("Authorization", "Bearer key-bearer")
	requireTrue(t, g.extractAPIKey(req) == "key-api", "x-api-key should win")

	// Bearer fallback with case-insensitive prefix.
	req = pipelineRequest("", "{}")
	req.Header.Set("Authorization", "bearer key-bearer")
	requireTrue(t, g.extractAPIKey(req) == "key-bearer", "bearer fallback failed")

	// Whitespace trimming.
	req = pipelineRequest(" key-space  ", "{}")
	requireTrue(t, g.extractAPIKey(req) == "key-space", "key should be trimmed")

	// Nothing present.
	req = pipelineRequest("", "{}")
	requireTrue(t, g.extractAPIKey(req) == "", "expected empty key")
}

// TestBuildPipelineKeys verifies the rate/budget key derivation.
func TestBuildPipelineKeys(t *testing.T) {
	g, _ := newGatewayForPipeline()
	cred := gatewayCredFixture()

	rateKey := g.buildRateLimitKey(cred, ResolvedUpstream{})
	requireTrue(t, rateKey == "cred:"+cred.ID.String(), "rate key mismatch: %s", rateKey)

	budgetKey := g.buildBudgetKey(cred, ResolvedUpstream{})
	requireTrue(t, budgetKey == "org:"+cred.OrganizationID.String(), "budget key mismatch: %s", budgetKey)
}

// estimateCostForFixture returns the expected cost for the fixture hints
// (1000 input, 2000 output): 0.01 + 0.06.
func estimateCostForFixture() float64 {
	return 1000.0/1000*0.01 + 2000.0/1000*0.03
}

// TestEstimateCost validates the token cost model.
func TestEstimateCost(t *testing.T) {
	g, _ := newGatewayForPipeline()

	cost := g.estimateCost(InspectorHints{EstimatedInputTokens: 1000, EstimatedOutputTokens: 2000})
	if math.Abs(cost-estimateCostForFixture()) > 1e-9 {
		t.Fatalf("cost mismatch: got %v, want %v", cost, estimateCostForFixture())
	}

	zero := g.estimateCost(InspectorHints{})
	requireTrue(t, zero == 0, "empty hints must cost zero")
}

// ---- Telemetry ---------------------------------------------------------------

// TestEmitTelemetry_UsageOnAllowed verifies allowed results emit a usage record
// populated from the credential and hints.
func TestEmitTelemetry_UsageOnAllowed(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	cred := gatewayCredFixture()

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	start := time.Now().Add(-42 * time.Millisecond)
	g.emitTelemetry(context.Background(), "req-1", res, start, nil)

	requireTrue(t, len(stubs.telemetry.usage) == 1, "want one usage record, got %d", len(stubs.telemetry.usage))
	requireTrue(t, len(stubs.telemetry.errs) == 0, "no error record expected")

	u := stubs.telemetry.usage[0]
	requireTrue(t, u.OrganizationID == cred.OrganizationID, "usage org mismatch")
	requireTrue(t, u.CredentialID == cred.ID, "usage credential mismatch")
	requireTrue(t, u.Model == "claude-sonnet-4-5", "usage model mismatch")
	requireTrue(t, u.InputTokens == 1000 && u.OutputTokens == 2000, "usage token counts mismatch")
	requireTrue(t, u.LatencyMs >= 42, "usage latency mismatch")
	requireTrue(t, u.StatusCode == http.StatusOK, "usage status mismatch")
}

// TestEmitTelemetry_ErrorOnDenied verifies denied results emit an error record.
func TestEmitTelemetry_ErrorOnDenied(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	stubs.rate.result = RateLimitResult{Allowed: false, Limit: 1000}

	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	start := time.Now().Add(-7 * time.Millisecond)
	g.emitTelemetry(context.Background(), "req-1", res, start, nil)

	requireTrue(t, len(stubs.telemetry.errs) == 1, "want one error record, got %d", len(stubs.telemetry.errs))
	requireTrue(t, len(stubs.telemetry.usage) == 0, "no usage record expected")

	e := stubs.telemetry.errs[0]
	requireTrue(t, e.ErrorType == "rate_limit_exceeded", "error type mismatch")
	requireTrue(t, e.StatusCode == 429, "error status mismatch")
	requireTrue(t, e.UpstreamError == "rate limit exceeded", "error message mismatch")
}

// TestEmitTelemetry_NilEmitterNoPanic verifies telemetry is optional.
func TestEmitTelemetry_NilEmitterNoPanic(t *testing.T) {
	g, _ := newGatewayForPipeline()
	g.telemetry = nil
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")
	start := time.Now().Add(-1 * time.Millisecond)
	g.emitTelemetry(context.Background(), "req-1", res, start, nil)
	// No telemetry component: nothing to assert, just no panic.
}

// ---- Lifecycle ---------------------------------------------------------------

// TestGateway_StartIdempotent verifies Start sets state once and is safe to
// call again.
func TestGateway_StartIdempotent(t *testing.T) {
	g, _ := newGatewayForPipeline()

	requireNoErr(t, g.Start(), "first Start")
	requireTrue(t, g.started, "started not set")
	requireTrue(t, !g.startTime.IsZero(), "startTime not set")

	first := g.startTime
	requireNoErr(t, g.Start(), "second Start")
	requireTrue(t, g.startTime == first, "second Start must be a no-op")
}

// TestGateway_HealthCheck verifies the health contract returns nil.
func TestGateway_HealthCheck(t *testing.T) {
	g, _ := newGatewayForPipeline()
	requireNoErr(t, g.HealthCheck(context.Background()), "HealthCheck")
}

// TestGateway_ShutdownDrainsProxyAndTelemetry verifies Shutdown closes the
// stream proxy and flushes/closes telemetry.
func TestGateway_ShutdownDrainsProxyAndTelemetry(t *testing.T) {
	g, stubs := newGatewayForPipeline()

	requireNoErr(t, g.Shutdown(5*time.Millisecond), "Shutdown")
	requireTrue(t, stubs.proxy.closeCalled, "stream proxy not closed")
	requireTrue(t, stubs.telemetry.flushCalled, "telemetry not flushed")
	requireTrue(t, stubs.telemetry.closeCalled, "telemetry not closed")
}

// TestGateway_Stats verifies the stats map with a real stream proxy (Stats
// asserts on *StreamProxyImpl).
func TestGateway_Stats(t *testing.T) {
	g := &Gateway{
		config:      DefaultGatewayConfig(),
		logger:      discardLogger(),
		streamProxy: NewStreamProxy(nil, discardLogger()),
	}
	requireNoErr(t, g.Start(), "Start")

	stats := g.Stats()
	requireTrue(t, stats["started"].(bool), "started flag wrong")
	requireTrue(t, stats["request_count"].(int64) == 0, "request count should be zero")
	requireTrue(t, stats["error_count"].(int64) == 0, "error count should be zero")
	requireTrue(t, stats["active_streams"].(int) == 0, "active streams should be zero")
	requireTrue(t, stats["uptime_seconds"].(float64) >= 0, "uptime should be non-negative")
}

// TestPipelineError_ErrorString verifies PipelineError formats stage + cause.
func TestPipelineError_ErrorString(t *testing.T) {
	err := &PipelineError{Stage: "authenticate", Err: errStub, StatusCode: 401, ErrorType: "authentication_error"}
	requireTrue(t, err.Error() == "authenticate: "+errStub.Error(), "message format wrong")
}
