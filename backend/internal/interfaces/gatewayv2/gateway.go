// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrRateLimited        = errors.New("rate limited")
	ErrBudgetExceeded     = errors.New("budget exceeded")
	ErrInternalError      = errors.New("internal error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrRequestTooLarge    = errors.New("request body too large")
)

// maxRequestBodyBytes bounds how much of a client request we buffer for
// inspection. The body must be read to extract routing hints, so the read is
// capped here: an oversized request is rejected before we allocate for it.
const maxRequestBodyBytes = 8 << 20 // 8 MiB

// Gateway is the main orchestrator wiring all components.
type Gateway struct {
	config GatewayConfig
	logger *slog.Logger

	// Components
	authenticator    Authenticator
	tenantResolver   TenantResolver
	policyResolver   PolicyResolver
	rateLimiter      RateLimiter
	budgetGuard      BudgetGuard
	requestInspector RequestInspector
	providerRouter   ProviderRouter
	streamProxy      StreamProxy
	telemetry        TelemetryEmitter
	activity         *ActivityTracker
	guardrails       *GuardrailEvaluator // Wrapper for the domain evaluator
	modelRepo        repositories.ModelRepository
	projectRepo      repositories.ProjectRepository

	// State
	mu           sync.RWMutex
	started      bool
	startTime    time.Time
	requestCount int64
	errorCount   int64
}

// GatewayDeps contains all dependencies for the gateway.
type GatewayDeps struct {
	Config GatewayConfig
	Logger *slog.Logger

	// Auth
	Hasher          ports.TokenHasher
	GatewayCredRepo repositories.GatewayCredentialRepository

	// Tenant
	ProviderAccounts    repositories.ProviderAccountRepository
	ProviderCredentials repositories.ProviderCredentialRepository
	Secrets             secrets.Store
	ModelRepo           repositories.ModelRepository
	ProjectRepo         repositories.ProjectRepository

	// Policy
	PolicyRepo repositories.PolicyRepository
	RuleRepo   repositories.PolicyRuleRepository
	Guardrails repositories.GuardrailsRepository

	// Usage accounting. The data plane derives usage and publishes it; the
	// worker owns every ledger write.
	EventBus ports.EventBus

	// Optional overrides
	CredentialCache *InMemoryCredentialCache
	UpstreamCache   *UpstreamCache
	PolicyCache     *PolicyCache
	RateLimiter     RateLimiter
	BudgetGuard     BudgetGuard
	Telemetry       TelemetryEmitter
	Activity        *ActivityTracker
}

// NewGateway creates a new gateway instance.
func NewGateway(deps GatewayDeps) (*Gateway, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	// Apply defaults
	config := deps.Config
	if config.RequestTimeout == 0 {
		config = DefaultGatewayConfig()
	}

	// Credential cache
	credCache := deps.CredentialCache
	if credCache == nil {
		credCache = NewInMemoryCredentialCache(10000, 5*time.Minute)
	}

	// Upstream cache
	upstreamCache := deps.UpstreamCache
	if upstreamCache == nil {
		upstreamCache = NewUpstreamCache(1000, 5*time.Minute)
	}

	// Policy cache
	policyCache := deps.PolicyCache
	if policyCache == nil {
		policyCache = NewPolicyCache(1000, 5*time.Minute)
	}

	// Rate limiter
	rateLimiter := deps.RateLimiter
	if rateLimiter == nil {
		rateLimiter = NewRateLimiter()
	}

	// Budget guard
	budgetGuard := deps.BudgetGuard
	if budgetGuard == nil {
		budgetGuard = NewBudgetGuardWithResolver(nil)
	}

	// Telemetry
	telemetry := deps.Telemetry
	if telemetry == nil {
		telemetry = NewTelemetryEmitter(10000, 500*time.Millisecond, deps.EventBus, deps.Logger)
	}

	// Build components
	authenticator := NewAuthenticator(deps.Hasher, deps.GatewayCredRepo, credCache)
	tenantResolver := NewTenantResolver(deps.ProviderAccounts, deps.ProviderCredentials, deps.Secrets)
	policyResolver := NewPolicyResolver(deps.PolicyRepo, deps.RuleRepo)

	// Update budget guard with policy resolver for snapshot sync
	if bg, ok := budgetGuard.(*BudgetGuardImpl); ok {
		bg.policyResolver = policyResolver
	}

	requestInspector := NewRequestInspector()
	providerRouter := NewProviderRouter("", "") // will use defaults
	streamProxy := NewStreamProxy(nil, deps.Logger)

	g := &Gateway{
		config:           config,
		logger:           deps.Logger,
		authenticator:    authenticator,
		tenantResolver:   tenantResolver,
		policyResolver:   policyResolver,
		rateLimiter:      rateLimiter,
		budgetGuard:      budgetGuard,
		requestInspector: requestInspector,
		providerRouter:   providerRouter,
		streamProxy:      streamProxy,
		telemetry:        telemetry,
		guardrails:       NewGuardrailEvaluator(deps.Guardrails),
		modelRepo:        deps.ModelRepo,
		projectRepo:      deps.ProjectRepo,
	}

	return g, nil
}

// ServeHTTP is the main entry point for the gateway.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := strings.TrimSpace(r.Header.Get("x-request-id"))
	if reqID == "" {
		reqID = strings.TrimSpace(r.Header.Get("x-correlation-id"))
	}
	if reqID == "" {
		reqID = uuid.New().String()
	}

	// Add request ID to response headers
	w.Header().Set("x-request-id", reqID)

	// Model listing endpoint
	if r.Method == http.MethodGet && (r.URL.Path == "/v1/models" || strings.HasSuffix(r.URL.Path, "/models")) {
		g.handleListModels(w, r, reqID)
		return
	}

	// Only handle POST /v1/messages and completion endpoints
	if r.Method != http.MethodPost {
		g.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	// Pipeline execution
	result := g.executePipeline(r.Context(), r, reqID)

	// An allowed request must actually reach the provider. Forward performs the
	// upstream call, streams the response back, and writes the status itself —
	// the gateway must never answer 200 merely because policy passed.
	if result.Allowed && result.Context != nil {
		userID := activityUserID(result.Context.Credential)
		if g.activity != nil {
			g.activity.Emit(domain.ActivityEvent{
				RequestID:      reqID,
				OrganizationID: result.Context.Credential.OrganizationID,
				UserID:         userID,
				TeamID:         result.Context.Credential.TeamID,
				ProjectID:      result.Context.Credential.ProjectID,
				Provider:       result.Context.Upstream.ProviderAccount.ProviderKind,
				Model:          result.Context.InspectorHints.Model,
				Status:         "pending",
				StartedAt:      start,
				CurrentState:   domain.ActivityStateActive,
			})
		}

		defer func() {
			if result.Context != nil && result.Context.BudgetRelease != nil {
				result.Context.BudgetRelease(0, errors.New("request terminated"))
			}
		}()

		observed, err := g.streamProxy.Forward(r.Context(), w, r, result.Context.UpstreamURL, result.Context.UpstreamHeaders)
		if err != nil {
			g.logger.Error("stream proxy forward failed", "error", err)
			if errors.Is(err, ErrUpstreamTimeout) {
				g.writeError(w, http.StatusGatewayTimeout, "upstream_timeout", "upstream provider timed out")
				if observed == nil {
					observed = &StreamUsage{}
				}
				observed.StatusCode = http.StatusGatewayTimeout
			} else if errors.Is(err, ErrUpstreamError) {
				g.writeError(w, http.StatusBadGateway, "upstream_error", "failed to connect to upstream provider")
				if observed == nil {
					observed = &StreamUsage{}
				}
				observed.StatusCode = http.StatusBadGateway
			} else if errors.Is(err, ErrClientDisconnected) {
				if observed == nil {
					observed = &StreamUsage{}
				}
				observed.StatusCode = 499
			} else if errors.Is(err, ErrStreamTruncated) {
				if observed == nil {
					observed = &StreamUsage{}
				}
				if observed.StatusCode == 0 || observed.StatusCode == 200 {
					observed.StatusCode = http.StatusBadGateway
				}
			}
		}
		g.emitTelemetry(r.Context(), reqID, result, start, observed)
		return
	}

	// Emit telemetry
	g.emitTelemetry(r.Context(), reqID, result, start, nil)

	// Write response
	g.writeResult(w, result)
}

// readAndRestoreBody buffers the request body once under a hard size bound and
// restores it, so inspection cannot leave an EOF behind for the upstream forward.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()

	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxRequestBodyBytes {
		return nil, ErrRequestTooLarge
	}

	// Restore the exact bytes so the forward carries the provider-native payload
	// untouched, and pin the length so a streamed request is not re-chunked.
	r.Body = io.NopCloser(bytes.NewReader(data))
	if r.ContentLength < 0 {
		r.ContentLength = int64(len(data))
	}
	return data, nil
}

// executePipeline runs the full gateway pipeline.
func (g *Gateway) executePipeline(ctx context.Context, r *http.Request, reqID string) PipelineResult {
	// Stage 1: Authenticate
	cred, err := g.authenticate(ctx, r)
	if err != nil {
		if errors.Is(err, ErrWrongScope) {
			return g.pipelineError("authenticate", err, http.StatusForbidden, "permission_error")
		}
		return g.pipelineError("authenticate", err, http.StatusUnauthorized, "authentication_error")
	}

	// Verify tenant and identity isolation: reject spoofed headers
	if clientOrg := r.Header.Get("X-Organization-ID"); clientOrg != "" {
		if clientOrg != cred.OrganizationID.String() {
			return g.pipelineError("authenticate", errors.New("cross-tenant header spoofing rejected"), http.StatusForbidden, "tenant_isolation_violation")
		}
	}
	if clientUser := r.Header.Get("X-User-ID"); clientUser != "" {
		if cred.SubjectType == "user" && clientUser != cred.SubjectID.String() {
			return g.pipelineError("authenticate", errors.New("user header spoofing rejected"), http.StatusForbidden, "permission_error")
		}
	}
	if clientProj := r.Header.Get("X-Project-ID"); clientProj != "" {
		if cred.ProjectID != nil && clientProj != cred.ProjectID.String() {
			return g.pipelineError("authenticate", errors.New("project header mismatch rejected"), http.StatusForbidden, "permission_error")
		}
		if cred.ProjectID == nil {
			if pID, err := uuid.Parse(clientProj); err == nil && g.projectRepo != nil {
				proj, err := g.projectRepo.Get(ctx, cred.OrganizationID, pID)
				if err != nil || proj.ID != pID {
					return g.pipelineError("authenticate", errors.New("unauthorized project header"), http.StatusForbidden, "permission_error")
				}
				cred.ProjectID = &pID
			}
		}
	}

	// Stage 2: Inspect request for routing hints and model.
	body, err := readAndRestoreBody(r)
	if err != nil {
		return g.pipelineError("request_inspector", err, http.StatusRequestEntityTooLarge, "request_too_large")
	}

	hints, err := g.requestInspector.Inspect(ctx, bytes.NewReader(body))
	if err != nil {
		g.logger.Warn("request inspection failed", "error", err)
		hints = InspectorHints{}
	}

	// Verify payload project_id integrity
	if hints.ProjectID != nil {
		if cred.ProjectID != nil && *hints.ProjectID != *cred.ProjectID {
			return g.pipelineError("authenticate", errors.New("payload project_id conflicts with credential project"), http.StatusForbidden, "permission_error")
		}
		if cred.ProjectID == nil && g.projectRepo != nil {
			proj, err := g.projectRepo.Get(ctx, cred.OrganizationID, *hints.ProjectID)
			if err != nil || proj.ID != *hints.ProjectID {
				return g.pipelineError("authenticate", errors.New("unauthorized payload project_id"), http.StatusForbidden, "permission_error")
			}
			cred.ProjectID = hints.ProjectID
		}
	}

	// Determine requested provider
	providerKind := hints.Provider
	if providerKind == "" {
		if p := r.Header.Get("x-provider"); p != "" {
			providerKind = strings.ToLower(strings.TrimSpace(p))
		}
	}
	if providerKind == "" && hints.Model != "" {
		switch {
		case strings.HasPrefix(hints.Model, "claude-"):
			providerKind = domain.ProviderKindAnthropic
		case strings.HasPrefix(hints.Model, "gpt-") || strings.HasPrefix(hints.Model, "o1") || strings.HasPrefix(hints.Model, "o3") || strings.HasPrefix(hints.Model, "text-embedding") || strings.HasPrefix(hints.Model, "dall-e"):
			providerKind = domain.ProviderKindOpenAI
		case strings.HasPrefix(hints.Model, "gemini-"):
			providerKind = domain.ProviderKindGemini
		}
	}
	if providerKind == "" {
		if strings.Contains(r.URL.Path, "/chat/completions") || strings.Contains(r.URL.Path, "/completions") || strings.Contains(r.URL.Path, "/embeddings") {
			providerKind = domain.ProviderKindOpenAI
		} else {
			providerKind = domain.ProviderKindAnthropic
		}
	}
	hints.Provider = providerKind

	// Stage 3: Resolve tenant/upstream for the specific provider (with fallback)
	upstream, err := g.tenantResolver.ResolveUpstreamForProvider(ctx, cred.OrganizationID, providerKind)
	if err != nil {
		upstream, err = g.tenantResolver.ResolveUpstream(ctx, cred.OrganizationID)
		if err != nil {
			return g.pipelineError("tenant_resolver", err, http.StatusServiceUnavailable, "upstream_unavailable")
		}
	}

	// Stage 4: Load policy snapshot (fail closed)
	policy, err := g.policyResolver.GetSnapshot(ctx, cred.OrganizationID)
	if err != nil {
		return g.pipelineError("policy_resolver", err, http.StatusServiceUnavailable, "policy_unavailable")
	}

	// Sync budget configurations from policy snapshot
	if bg, ok := g.budgetGuard.(*BudgetGuardImpl); ok {
		bg.SyncFromSnapshot(policy)
	}

	reqCtx := &RequestContext{
		Credential:     cred,
		Upstream:       upstream,
		PolicySnapshot: policy,
		InspectorHints: hints,
		StartTime:      time.Now(),
	}

	// Stage 4.5: Guardrails
	if g.guardrails != nil {
		gResult := g.guardrails.CheckInput(ctx, body, cred.OrganizationID.String(), reqID, cred)
		if gResult.Triggered && gResult.Action == domain.GuardrailActionBlock {
			res := g.pipelineError("guardrails", errors.New("blocked by guardrail: "+gResult.MatchedCategory), http.StatusForbidden, "guardrail_blocked")
			res.Context = reqCtx
			return res
		}
	}

	// Stage 5: Authorize the requested model against policy.
	if allowed, reason := policy.Authorize(hints, cred); !allowed {
		res := g.pipelineError("policy_authorizer", errors.New(reason), http.StatusForbidden, "policy_denied")
		res.Context = reqCtx
		return res
	}

	// Validate model is supported
	if hints.Model != "" && !g.isModelSupported(ctx, hints.Model) {
		res := g.pipelineError("model_resolver", fmt.Errorf("model %q is not supported", hints.Model), http.StatusBadRequest, "unsupported_model")
		res.Context = reqCtx
		return res
	}

	// Stage 6: Rate limiting across all applicable scopes
	applicableRLs := policy.applicableRateLimits(cred)
	var finalRateLimitResult RateLimitResult
	if len(applicableRLs) == 0 {
		rateKey := g.buildRateLimitKey(cred, upstream)
		rateLimitCfg, rateLimitWindow := g.config.DefaultRateLimit, g.config.DefaultRateWindow
		if cfg, ok := policy.rateLimitFor("credential"); ok {
			rateLimitCfg, rateLimitWindow = cfg.Limit, cfg.Window
		}
		finalRateLimitResult = g.rateLimiter.Check(ctx, rateKey, rateLimitCfg, rateLimitWindow)
		if !finalRateLimitResult.Allowed {
			return g.rateLimitedResult(finalRateLimitResult, reqCtx)
		}
	} else {
		for _, cfg := range applicableRLs {
			rlKey := g.buildRateLimitScopedKey(cred, cfg)
			rlResult := g.rateLimiter.Check(ctx, rlKey, cfg.Limit, cfg.Window)
			if !rlResult.Allowed {
				return g.rateLimitedResult(rlResult, reqCtx)
			}
			finalRateLimitResult = rlResult
		}
	}

	// Stage 7: Budget check across all applicable scopes (spend and token budgets)
	// Stage 7: Budgets (atomic reservation preventing concurrent overspend)
	applicableBudgets := policy.applicableBudgets(cred)
	var finalBudgetResult BudgetResult
	var releaseFuncs []BudgetReleaseFunc

	if len(applicableBudgets) == 0 {
		budgetKey := g.buildBudgetKey(cred, upstream)
		estimatedCost := g.estimateCost(hints)
		budgetCfg, budgetWindow := g.config.DefaultBudgetLimit, g.config.DefaultBudgetWindow
		if cfg, ok := policy.budgetFor("organization"); ok {
			budgetCfg, budgetWindow = cfg.Limit, cfg.Window
		}
		bResult, release := g.budgetGuard.Reserve(ctx, budgetKey, budgetCfg, budgetWindow, estimatedCost, reqID)
		if !bResult.Allowed {
			return g.budgetExceededResult(bResult, reqCtx)
		}
		finalBudgetResult = bResult
		releaseFuncs = append(releaseFuncs, release)
	} else {
		for _, cfg := range applicableBudgets {
			budgetKey := cfg.Key
			if budgetKey == "" {
				budgetKey = cfg.Scope
			}
			var checkCost float64
			if cfg.LimitType == "tokens" {
				checkCost = float64(hints.EstimatedInputTokens + hints.EstimatedOutputTokens)
				if checkCost <= 0 {
					checkCost = 1
				}
			} else {
				checkCost = g.estimateCost(hints)
			}
			bResult, release := g.budgetGuard.Reserve(ctx, budgetKey, cfg.Limit, cfg.Window, checkCost, reqID)
			if !bResult.Allowed {
				// Rollback already acquired reservations for this request
				for _, rFn := range releaseFuncs {
					rFn(0, errors.New("budget_exceeded"))
				}
				return g.budgetExceededResult(bResult, reqCtx)
			}
			finalBudgetResult = bResult
			releaseFuncs = append(releaseFuncs, release)
		}
	}

	combinedRelease := func(actualCost float64, err error) {
		for _, rFn := range releaseFuncs {
			rFn(actualCost, err)
		}
	}
	reqCtx.BudgetRelease = combinedRelease

	// Stage 8: Route to provider
	upstreamURL, upstreamHeaders, err := g.providerRouter.Route(ctx, upstream, hints)
	if err != nil {
		res := g.pipelineError("provider_router", err, http.StatusInternalServerError, "routing_error")
		res.Context = reqCtx
		return res
	}

	// Enrich request context for downstream
	reqCtx.RateLimit = finalRateLimitResult
	reqCtx.BudgetResult = finalBudgetResult
	reqCtx.UpstreamURL = upstreamURL
	reqCtx.UpstreamHeaders = upstreamHeaders

	return PipelineResult{
		Allowed:    true,
		Context:    reqCtx,
		StatusCode: http.StatusOK,
		// UpstreamURL and Headers will be used by stream proxy
	}
}

// authenticate extracts and validates the API key.
func (g *Gateway) authenticate(ctx context.Context, r *http.Request) (GatewayCredential, error) {
	rawKey := g.extractAPIKey(r)
	if rawKey == "" {
		return GatewayCredential{}, ErrUnauthorized
	}

	cred, err := g.authenticator.Authenticate(ctx, rawKey)
	if err != nil {
		return GatewayCredential{}, err
	}

	return cred, nil
}

// extractAPIKey extracts the API key from request headers.
func (g *Gateway) extractAPIKey(r *http.Request) string {
	// Try x-api-key first (Claude Code standard)
	if key := r.Header.Get("x-api-key"); key != "" {
		return strings.TrimSpace(key)
	}

	// Fallback to Authorization: Bearer
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}

	return ""
}

// buildRateLimitKey constructs the fallback rate limit key from credential and upstream.
func (g *Gateway) buildRateLimitKey(cred GatewayCredential, upstream ResolvedUpstream) string {
	return fmt.Sprintf("org:%s:cred:%s", cred.OrganizationID, cred.ID)
}

// buildRateLimitScopedKey constructs a strictly tenant-isolated and actor-bound rate limit key.
func (g *Gateway) buildRateLimitScopedKey(cred GatewayCredential, cfg RateLimitConfig) string {
	switch cfg.Scope {
	case "organization", "org", "":
		return "org:" + cred.OrganizationID.String()
	case "user":
		if cred.SubjectType == "user" {
			return fmt.Sprintf("org:%s:user:%s", cred.OrganizationID, cred.SubjectID)
		}
		return "org:" + cred.OrganizationID.String()
	case "service_account":
		if cred.SubjectType == "service_account" {
			return fmt.Sprintf("org:%s:sa:%s", cred.OrganizationID, cred.SubjectID)
		}
		return "org:" + cred.OrganizationID.String()
	case "team":
		if cred.TeamID != nil {
			return fmt.Sprintf("org:%s:team:%s", cred.OrganizationID, cred.TeamID)
		}
		return "org:" + cred.OrganizationID.String()
	case "project":
		if cred.ProjectID != nil {
			return fmt.Sprintf("org:%s:proj:%s", cred.OrganizationID, cred.ProjectID)
		}
		return "org:" + cred.OrganizationID.String()
	case "credential", "gateway_credential":
		return fmt.Sprintf("org:%s:cred:%s", cred.OrganizationID, cred.ID)
	default:
		if cfg.Key != "" {
			if strings.HasPrefix(cfg.Key, "org:") {
				return cfg.Key
			}
			return fmt.Sprintf("org:%s:%s", cred.OrganizationID, cfg.Key)
		}
		return "org:" + cred.OrganizationID.String()
	}
}

// buildBudgetKey constructs the budget key from credential and upstream.
// The key format matches the budget rule key: scope:scope_id
func (g *Gateway) buildBudgetKey(cred GatewayCredential, upstream ResolvedUpstream) string {
	// Budget by credential scope - supports organization, team, project, service_account
	if cred.TeamID != nil {
		return "team:" + cred.TeamID.String()
	}
	if cred.ProjectID != nil {
		return "project:" + cred.ProjectID.String()
	}
	if cred.SubjectType == "service_account" {
		return "service_account:" + cred.SubjectID.String()
	}
	// Default to organization scope
	return "org:" + cred.OrganizationID.String()
}

// estimateCost provides a rough cost estimate for the request.
func (g *Gateway) estimateCost(hints InspectorHints) float64 {
	// Rough estimate: $0.01 per 1K input tokens, $0.03 per 1K output tokens
	inputCost := float64(hints.EstimatedInputTokens) / 1000 * 0.01
	outputCost := float64(hints.EstimatedOutputTokens) / 1000 * 0.03
	return inputCost + outputCost
}

// rateLimitedResult creates a rate limited response.
func (g *Gateway) rateLimitedResult(rl RateLimitResult, reqCtx ...*RequestContext) PipelineResult {
	var ctx *RequestContext
	if len(reqCtx) > 0 && reqCtx[0] != nil {
		ctx = reqCtx[0]
		ctx.RateLimit = rl
	} else {
		ctx = &RequestContext{RateLimit: rl}
	}
	return PipelineResult{
		Allowed:      false,
		StatusCode:   429,
		ErrorType:    "rate_limit_exceeded",
		ErrorMessage: "rate limit exceeded",
		Context:      ctx,
	}
}

// budgetExceededResult creates a budget exceeded response.
func (g *Gateway) budgetExceededResult(br BudgetResult, reqCtx ...*RequestContext) PipelineResult {
	var ctx *RequestContext
	if len(reqCtx) > 0 && reqCtx[0] != nil {
		ctx = reqCtx[0]
		ctx.BudgetResult = br
	} else {
		ctx = &RequestContext{BudgetResult: br}
	}
	return PipelineResult{
		Allowed:      false,
		StatusCode:   402,
		ErrorType:    "budget_exceeded",
		ErrorMessage: "budget exceeded",
		Context:      ctx,
	}
}

// pipelineError creates a generic pipeline error result.
func (g *Gateway) pipelineError(stage string, err error, status int, errType string) PipelineResult {
	g.logger.Error("pipeline error", "stage", stage, "error", err)
	atomic.AddInt64(&g.errorCount, 1)

	return PipelineResult{
		Allowed:      false,
		StatusCode:   status,
		ErrorType:    errType,
		ErrorMessage: err.Error(),
	}
}

// writeResult writes a denied pipeline result to the response. Allowed results
// never reach here: ServeHTTP forwards them upstream and writes the response
// from the provider, so this path is deny-only by design.
func (g *Gateway) writeResult(w http.ResponseWriter, result PipelineResult) {
	if result.Allowed {
		g.logger.Error("writeResult called for an allowed result; upstream response was not written")
		return
	}

	if result.StatusCode == http.StatusTooManyRequests && result.Context != nil {
		rl := result.Context.RateLimit
		if rl.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(rl.RetryAfter.Seconds()))))
		}
		if rl.Limit > 0 {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.Limit))
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rl.Remaining))
		if !rl.ResetAt.IsZero() {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rl.ResetAt.Unix(), 10))
		}
	}

	// Error response in provider-native format
	g.writeError(w, result.StatusCode, result.ErrorType, result.ErrorMessage)
}

// writeError writes an error response in Anthropic-compatible format.
func (g *Gateway) writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	body := anthropicErrorBody{
		Type: "error",
		Error: struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    errType,
			Message: message,
		},
	}

	// Fast path: encode directly
	_ = json.NewEncoder(w).Encode(body)
}

// emitTelemetry emits usage and error telemetry.
func (g *Gateway) emitTelemetry(ctx context.Context, reqID string, result PipelineResult, start time.Time, observed *StreamUsage) {
	completedAt := time.Now().UTC()
	latencyMs := completedAt.Sub(start).Milliseconds()
	startUTC := start.UTC()

	var providerKind, model string
	if result.Context != nil {
		providerKind = result.Context.Upstream.ProviderAccount.ProviderKind
		model = result.Context.InspectorHints.Model

		if g.activity != nil {
			state := domain.ActivityStateCompleted
			if (observed != nil && observed.StatusCode >= 400) || result.StatusCode >= 400 {
				state = domain.ActivityStateFailed
			}
			g.activity.Emit(domain.ActivityEvent{
				RequestID:      reqID,
				OrganizationID: result.Context.Credential.OrganizationID,
				UserID:         activityUserID(result.Context.Credential),
				TeamID:         result.Context.Credential.TeamID,
				ProjectID:      result.Context.Credential.ProjectID,
				Provider:       providerKind,
				Model:          model,
				Status:         result.ErrorType,
				StartedAt:      startUTC,
				CurrentState:   state,
			})
		}
	}

	if g.telemetry == nil {
		return
	}

	if result.Allowed && result.Context != nil {
		actualStatusCode := result.StatusCode
		if observed != nil && observed.StatusCode != 0 {
			actualStatusCode = observed.StatusCode
		}

		usage := UsageRecord{
			RequestID:        reqID,
			OrganizationID:   result.Context.Credential.OrganizationID,
			CredentialID:     result.Context.Credential.ID,
			SubjectType:      result.Context.Credential.SubjectType,
			SubjectID:        result.Context.Credential.SubjectID,
			ProjectID:        result.Context.Credential.ProjectID,
			TeamID:           result.Context.Credential.TeamID,
			Model:            result.Context.InspectorHints.Model,
			LatencyMs:        latencyMs,
			StatusCode:       actualStatusCode,
			Timestamp:        completedAt,
			RequestStartedAt: startUTC,
			CompletedAt:      completedAt,
		}

		totalLatencyMs := int32(latencyMs)
		usage.TotalLatencyMs = &totalLatencyMs

		upstreamProvider := result.Context.Upstream.ProviderAccount.ProviderKind
		if upstreamProvider == "" && observed != nil {
			upstreamProvider = observed.Provider
		}
		if upstreamProvider == "" {
			upstreamProvider = "unknown"
		}

		// Provider-reported counts are authoritative; fallback to estimates when provider does not supply counts.
		if observed != nil && (observed.InputTokens > 0 || observed.OutputTokens > 0) {
			usage.InputTokens = observed.InputTokens
			usage.OutputTokens = observed.OutputTokens
			usage.CacheReadTokens = observed.CacheReadTokens
			usage.CacheWriteTokens = observed.CacheWriteTokens
			meta := map[string]any{"provider": upstreamProvider, "tokens_reported": true, "estimated": false}
			if observed.ReasoningTokens > 0 {
				meta["reasoning_tokens"] = observed.ReasoningTokens
			}
			usage.ProviderMeta = meta

			if !observed.FirstTokenAt.IsZero() {
				firstToken := observed.FirstTokenAt.UTC()
				usage.FirstTokenAt = &firstToken
				ttft := int32(firstToken.Sub(startUTC).Milliseconds())
				usage.TimeToFirstTokenMs = &ttft
			}
		} else if actualStatusCode >= 400 {
			usage.InputTokens = 0
			usage.OutputTokens = 0
			usage.ProviderMeta = map[string]any{"provider": upstreamProvider, "tokens_reported": false, "estimated": false}
		} else {
			// Fallback to estimates, explicitly flagged as estimated
			usage.InputTokens = result.Context.InspectorHints.EstimatedInputTokens
			usage.OutputTokens = result.Context.InspectorHints.EstimatedOutputTokens
			usage.ProviderMeta = map[string]any{"provider": upstreamProvider, "tokens_reported": false, "estimated": true}
		}

		// Consistent initial cost calculation in micro-dollars ($0.01/1k input = 10 micros/token, $0.03/1k output = 30 micros/token).
		// Exact cost is backfilled during rollup using the organization's versioned price sheet.
		cost := (float64(usage.InputTokens) / 1000.0 * 0.01) + (float64(usage.OutputTokens) / 1000.0 * 0.03)
		costMicros := int64(math.Round(cost * 1000000.0))
		if costMicros <= 0 && (usage.InputTokens > 0 || usage.OutputTokens > 0) {
			costMicros = int64(usage.InputTokens*10 + usage.OutputTokens*30)
		}
		usage.CostMicros = costMicros

		allowedDecision := domain.PolicyDecisionAllowed
		usage.PolicyDecision = &allowedDecision
		if observed != nil && observed.ReasoningTokens > 0 {
			usage.ReasoningTokens = observed.ReasoningTokens
		}
		if actualStatusCode >= 400 {
			usage.CostStatus = domain.CostStatusUnavailable
			usage.CostMicros = 0
		} else if costMicros > 0 {
			if observed != nil && (observed.InputTokens > 0 || observed.OutputTokens > 0) {
				usage.CostStatus = domain.CostStatusCalculated
			} else {
				usage.CostStatus = domain.CostStatusEstimated
			}
		} else if usage.InputTokens > 0 || usage.OutputTokens > 0 {
			usage.CostStatus = domain.CostStatusUnavailable
		} else {
			usage.CostStatus = domain.CostStatusFree
		}

		// Record and finalize budget reservation:
		// Actual tokens are charged for successful requests (status < 400) or client disconnects (499) with partial tokens.
		// Failed/rejected requests release their reservation with zero charge.
		if actualStatusCode < 400 || (actualStatusCode == 499 && (usage.InputTokens > 0 || usage.OutputTokens > 0)) {
			if result.Context != nil && result.Context.BudgetRelease != nil {
				result.Context.BudgetRelease(cost, nil)
			} else {
				appBudgets := result.Context.PolicySnapshot.applicableBudgets(result.Context.Credential)
				if len(appBudgets) == 0 {
					budgetKey := g.buildBudgetKey(result.Context.Credential, result.Context.Upstream)
					if err := g.budgetGuard.RecordUsage(ctx, budgetKey, cost); err != nil {
						g.logger.Warn("failed to record local budget usage", "error", err, "key", budgetKey)
					}
				} else {
					for _, cfg := range appBudgets {
						usageAmt := cost
						if cfg.LimitType == "tokens" {
							usageAmt = float64(usage.InputTokens + usage.OutputTokens)
						}
						if err := g.budgetGuard.RecordUsage(ctx, cfg.Key, usageAmt); err != nil {
							g.logger.Warn("failed to record local budget usage", "error", err, "key", cfg.Key)
						}
					}
				}
			}
		} else {
			if result.Context != nil && result.Context.BudgetRelease != nil {
				result.Context.BudgetRelease(0, fmt.Errorf("request failed with status %d", actualStatusCode))
			}
		}

		g.telemetry.EmitUsage(ctx, usage)
	} else {
		// Error telemetry
		orgID := uuid.Nil
		credID := uuid.Nil
		var projID, teamID *uuid.UUID
		var subType string
		var subID uuid.UUID
		if result.Context != nil {
			orgID = result.Context.Credential.OrganizationID
			credID = result.Context.Credential.ID
			projID = result.Context.Credential.ProjectID
			teamID = result.Context.Credential.TeamID
			subType = result.Context.Credential.SubjectType
			subID = result.Context.Credential.SubjectID
		}

		errRecord := ErrorRecord{
			OrganizationID: orgID,
			CredentialID:   credID,
			ErrorType:      result.ErrorType,
			ErrorCode:      result.ErrorType,
			StatusCode:     result.StatusCode,
			LatencyMs:      latencyMs,
			Timestamp:      time.Now(),
			UpstreamError:  result.ErrorMessage,
		}
		g.telemetry.EmitError(ctx, errRecord)

		if orgID != uuid.Nil {
			provKind := providerKind
			if provKind == "" {
				provKind = "unknown"
			}
			deniedDecision := domain.PolicyDecisionDenied
			denialReason := result.ErrorType
			if denialReason == "" {
				denialReason = result.ErrorMessage
			}
			failedUsage := UsageRecord{
				RequestID:        reqID,
				OrganizationID:   orgID,
				CredentialID:     credID,
				SubjectType:      subType,
				SubjectID:        subID,
				ProjectID:        projID,
				TeamID:           teamID,
				Model:            model,
				LatencyMs:        latencyMs,
				StatusCode:       result.StatusCode,
				Timestamp:        completedAt,
				RequestStartedAt: startUTC,
				CompletedAt:      completedAt,
				PolicyDecision:   &deniedDecision,
				DenialReason:     &denialReason,
				CostStatus:       domain.CostStatusUnavailable,
				ProviderMeta:     map[string]any{"provider": provKind},
			}
			g.telemetry.EmitUsage(ctx, failedUsage)
		}
	}
}

// Stats returns gateway statistics.
func (g *Gateway) Stats() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]any{
		"uptime_seconds": time.Since(g.startTime).Seconds(),
		"request_count":  atomic.LoadInt64(&g.requestCount),
		"error_count":    atomic.LoadInt64(&g.errorCount),
		"active_streams": g.streamProxy.(*StreamProxyImpl).ActiveStreamCount(),
		"started":        g.started,
		"start_time":     g.startTime,
	}
}

// HealthCheck returns the gateway health status.
func (g *Gateway) HealthCheck(ctx context.Context) error {
	// Check critical dependencies
	// Could check DB, Redis, upstream connectivity, etc.
	return nil
}

// Shutdown gracefully shuts down the gateway.
func (g *Gateway) Shutdown(gracePeriod time.Duration) error {
	g.logger.Info("shutting down gateway", "grace_period", gracePeriod)

	// Stop accepting new requests (handled by HTTP server shutdown)

	// Wait for active streams to drain
	if err := g.streamProxy.Close(gracePeriod); err != nil {
		g.logger.Error("stream proxy shutdown error", "error", err)
	}

	// Flush telemetry
	if err := g.telemetry.Flush(context.Background()); err != nil {
		g.logger.Error("telemetry flush error", "error", err)
	}

	// Close telemetry
	if err := g.telemetry.Close(); err != nil {
		g.logger.Error("telemetry close error", "error", err)
	}

	g.logger.Info("gateway shutdown complete")
	return nil
}

func activityUserID(cred GatewayCredential) *uuid.UUID {
	if cred.SubjectType == string(domain.SubjectTypeUser) {
		return &cred.SubjectID
	}
	return nil
}

// anthropicErrorBody is the provider-native error envelope.
type anthropicErrorBody struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Start initializes the gateway.
func (g *Gateway) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return nil
	}

	g.startTime = time.Now()
	g.started = true
	g.logger.Info("gateway started", "config", g.config)
	return nil
}

// InvalidatePolicy immediately evicts the cached policy snapshot for an organization.
func (g *Gateway) InvalidatePolicy(orgID uuid.UUID) {
	if g.policyResolver != nil {
		g.policyResolver.Invalidate(orgID)
	}
}

// InvalidateCredentialByID immediately evicts a cached credential by ID.
func (g *Gateway) InvalidateCredentialByID(ctx context.Context, id string) {
	if authImpl, ok := g.authenticator.(*AuthenticatorImpl); ok {
		authImpl.InvalidateByID(ctx, id)
	}
}

// handleListModels serves GET /v1/models, returning permitted models for the authenticated credential.
func (g *Gateway) handleListModels(w http.ResponseWriter, r *http.Request, reqID string) {
	cred, err := g.authenticate(r.Context(), r)
	if err != nil {
		if errors.Is(err, ErrWrongScope) {
			g.writeError(w, http.StatusForbidden, "permission_error", err.Error())
			return
		}
		g.writeError(w, http.StatusUnauthorized, "authentication_error", "invalid credential")
		return
	}

	policy, err := g.policyResolver.GetSnapshot(r.Context(), cred.OrganizationID)
	if err != nil {
		g.writeError(w, http.StatusServiceUnavailable, "policy_unavailable", "failed to load policy")
		return
	}

	type modelEntry struct {
		ID          string `json:"id"`
		Object      string `json:"object"`
		Created     int64  `json:"created"`
		OwnedBy     string `json:"owned_by"`
		Provider    string `json:"provider"`
		DisplayName string `json:"display_name,omitempty"`
	}

	var allModels []modelEntry
	if g.modelRepo != nil {
		if dbModels, err := g.modelRepo.List(r.Context()); err == nil && len(dbModels) > 0 {
			for _, m := range dbModels {
				allModels = append(allModels, modelEntry{
					ID:          m.ModelSlug,
					Object:      "model",
					Created:     m.CreatedAt.Unix(),
					OwnedBy:     m.ProviderKind,
					Provider:    m.ProviderKind,
					DisplayName: m.DisplayName,
				})
			}
		}
	}

	if len(allModels) == 0 {
		// Standard default catalog across supported providers
		allModels = []modelEntry{
			{ID: "claude-sonnet-4-5", Object: "model", Created: 1700000000, OwnedBy: "anthropic", Provider: "anthropic", DisplayName: "Claude 3.5 Sonnet"},
			{ID: "claude-3-5-sonnet-20241022", Object: "model", Created: 1700000000, OwnedBy: "anthropic", Provider: "anthropic", DisplayName: "Claude 3.5 Sonnet (20241022)"},
			{ID: "claude-3-haiku-20240307", Object: "model", Created: 1700000000, OwnedBy: "anthropic", Provider: "anthropic", DisplayName: "Claude 3 Haiku"},
			{ID: "claude-3-opus-20240229", Object: "model", Created: 1700000000, OwnedBy: "anthropic", Provider: "anthropic", DisplayName: "Claude 3 Opus"},
			{ID: "gpt-4o", Object: "model", Created: 1700000000, OwnedBy: "openai", Provider: "openai", DisplayName: "GPT-4o"},
			{ID: "gpt-4o-mini", Object: "model", Created: 1700000000, OwnedBy: "openai", Provider: "openai", DisplayName: "GPT-4o Mini"},
			{ID: "o1", Object: "model", Created: 1700000000, OwnedBy: "openai", Provider: "openai", DisplayName: "o1"},
			{ID: "o3-mini", Object: "model", Created: 1700000000, OwnedBy: "openai", Provider: "openai", DisplayName: "o3 Mini"},
			{ID: "text-embedding-3-small", Object: "model", Created: 1700000000, OwnedBy: "openai", Provider: "openai", DisplayName: "Text Embedding 3 Small"},
			{ID: "gemini-1.5-pro", Object: "model", Created: 1700000000, OwnedBy: "gemini", Provider: "gemini", DisplayName: "Gemini 1.5 Pro"},
			{ID: "gemini-1.5-flash", Object: "model", Created: 1700000000, OwnedBy: "gemini", Provider: "gemini", DisplayName: "Gemini 1.5 Flash"},
			{ID: "gemini-2.5-flash", Object: "model", Created: 1700000000, OwnedBy: "gemini", Provider: "gemini", DisplayName: "Gemini 2.5 Flash"},
			{ID: "gemini-2.5-pro", Object: "model", Created: 1700000000, OwnedBy: "gemini", Provider: "gemini", DisplayName: "Gemini 2.5 Pro"},
		}
	}

	var permitted []modelEntry
	for _, m := range allModels {
		allowed, _ := policy.Authorize(InspectorHints{Model: m.ID, Provider: m.Provider}, cred)
		if allowed {
			permitted = append(permitted, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   permitted,
		"items":  permitted,
	})
}

// isModelSupported checks whether a requested model matches known provider patterns or catalog entries.
func (g *Gateway) isModelSupported(ctx context.Context, model string) bool {
	if model == "" {
		return true
	}
	if strings.HasPrefix(model, "claude-") ||
		strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "text-embedding") || strings.HasPrefix(model, "dall-e") ||
		strings.HasPrefix(model, "gemini-") {
		return true
	}

	if g.modelRepo != nil {
		if models, err := g.modelRepo.List(ctx); err == nil {
			for _, m := range models {
				if m.ModelSlug == model || m.DisplayName == model {
					return true
				}
			}
		}
	}
	return false
}
