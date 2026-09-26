// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrRateLimited      = errors.New("rate limited")
	ErrBudgetExceeded   = errors.New("budget exceeded")
	ErrInternalError    = errors.New("internal error")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// Gateway is the main orchestrator wiring all components.
type Gateway struct {
	config          GatewayConfig
	logger          *slog.Logger

	// Components
	authenticator   Authenticator
	tenantResolver  TenantResolver
	policyResolver  PolicyResolver
	rateLimiter     RateLimiter
	budgetGuard     BudgetGuard
	requestInspector RequestInspector
	providerRouter  ProviderRouter
	streamProxy     StreamProxy
	telemetry       TelemetryEmitter

	// State
	mu           sync.RWMutex
	started      bool
	startTime    time.Time
	requestCount int64
	errorCount   int64
}

// GatewayDeps contains all dependencies for the gateway.
type GatewayDeps struct {
	Config            GatewayConfig
	Logger            *slog.Logger

	// Auth
	Hasher            ports.TokenHasher
	GatewayCredRepo   repositories.GatewayCredentialRepository

	// Tenant
	ProviderAccounts    repositories.ProviderAccountRepository
	ProviderCredentials repositories.ProviderCredentialRepository
	Secrets             secrets.Store

	// Policy
	PolicyRepo repositories.PolicyRepository
	RuleRepo   repositories.PolicyRuleRepository

	// Optional overrides
	CredentialCache *InMemoryCredentialCache
	UpstreamCache   *UpstreamCache
	PolicyCache     *PolicyCache
	RateLimiter     RateLimiter
	BudgetGuard     BudgetGuard
	Telemetry       TelemetryEmitter
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
		budgetGuard = NewBudgetGuard()
	}

	// Telemetry
	telemetry := deps.Telemetry
	if telemetry == nil {
		telemetry = NewTelemetryEmitter(10000, 5*time.Second, deps.Logger)
	}

	// Build components
	authenticator := NewAuthenticator(deps.Hasher, deps.GatewayCredRepo, credCache)
	tenantResolver := NewTenantResolver(deps.ProviderAccounts, deps.ProviderCredentials, deps.Secrets)
	policyResolver := NewPolicyResolver(deps.PolicyRepo, deps.RuleRepo)
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
	}

	return g, nil
}

// ServeHTTP is the main entry point for the gateway.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := uuid.New().String()

	// Add request ID to response headers
	w.Header().Set("x-request-id", reqID)

	// Only handle POST /v1/messages and related endpoints
	if r.Method != http.MethodPost {
		g.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	// Pipeline execution
	result := g.executePipeline(r.Context(), r, reqID)

	// Emit telemetry
	latencyMs := time.Since(start).Milliseconds()
	g.emitTelemetry(r.Context(), result, latencyMs)

	// Write response
	g.writeResult(w, result)
}

// executePipeline runs the full gateway pipeline.
func (g *Gateway) executePipeline(ctx context.Context, r *http.Request, reqID string) PipelineResult {
	// Stage 1: Authenticate
	cred, err := g.authenticate(ctx, r)
	if err != nil {
		return g.pipelineError("authenticate", err, http.StatusUnauthorized, "authentication_error")
	}

	// Stage 2: Resolve tenant/upstream
	upstream, err := g.tenantResolver.ResolveUpstream(ctx, cred.OrganizationID)
	if err != nil {
		return g.pipelineError("tenant_resolver", err, http.StatusServiceUnavailable, "upstream_unavailable")
	}

	// Stage 3: Load policy snapshot (fail closed: an unavailable policy denies
	// the request instead of degrading to defaults).
	policy, err := g.policyResolver.GetSnapshot(ctx, cred.OrganizationID)
	if err != nil {
		return g.pipelineError("policy_resolver", err, http.StatusServiceUnavailable, "policy_unavailable")
	}

	// Stage 4: Inspect request for routing hints
	hints, err := g.requestInspector.Inspect(ctx, r.Body)
	if err != nil {
		g.logger.Warn("request inspection failed", "error", err)
		// Continue with empty hints
		hints = InspectorHints{}
	}

	// Stage 5: Authorize the requested model against policy. This runs before
	// rate/budget so denied requests consume no quota.
	if allowed, reason := policy.Authorize(hints); !allowed {
		return g.pipelineError("policy_authorizer", errors.New(reason), http.StatusForbidden, "policy_denied")
	}

	// Stage 6: Rate limiting (snapshot config when present, else defaults)
	rateKey := g.buildRateLimitKey(cred, upstream)
	rateLimitCfg, rateLimitWindow := g.config.DefaultRateLimit, g.config.DefaultRateWindow
	if cfg, ok := policy.rateLimitFor("credential"); ok {
		rateLimitCfg, rateLimitWindow = cfg.Limit, cfg.Window
	}
	rateLimitResult := g.rateLimiter.Check(ctx, rateKey, rateLimitCfg, rateLimitWindow)
	if !rateLimitResult.Allowed {
		return g.rateLimitedResult(rateLimitResult)
	}

	// Stage 7: Budget check (snapshot config when present, else defaults)
	budgetKey := g.buildBudgetKey(cred, upstream)
	estimatedCost := g.estimateCost(hints)
	budgetCfg, budgetWindow := g.config.DefaultBudgetLimit, g.config.DefaultBudgetWindow
	if cfg, ok := policy.budgetFor("organization"); ok {
		budgetCfg, budgetWindow = cfg.Limit, cfg.Window
	}
	budgetResult := g.budgetGuard.Check(ctx, budgetKey, budgetCfg, budgetWindow, estimatedCost)
	if !budgetResult.Allowed {
		return g.budgetExceededResult(budgetResult)
	}

	// Stage 8: Route to provider
	upstreamURL, upstreamHeaders, err := g.providerRouter.Route(ctx, upstream, hints)
	if err != nil {
		return g.pipelineError("provider_router", err, http.StatusInternalServerError, "routing_error")
	}

	// Build request context for downstream
	reqCtx := &RequestContext{
		Credential:      cred,
		Upstream:        upstream,
		PolicySnapshot:  policy,
		RateLimit:       rateLimitResult,
		BudgetResult:    budgetResult,
		InspectorHints:  hints,
		StartTime:       time.Now(),
		UpstreamURL:     upstreamURL,
		UpstreamHeaders: upstreamHeaders,
	}

	return PipelineResult{
		Allowed: true,
		Context: reqCtx,
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

// buildRateLimitKey constructs the rate limit key from credential and upstream.
func (g *Gateway) buildRateLimitKey(cred GatewayCredential, upstream ResolvedUpstream) string {
	// Rate limit by credential ID
	return "cred:" + cred.ID.String()
}

// buildBudgetKey constructs the budget key.
func (g *Gateway) buildBudgetKey(cred GatewayCredential, upstream ResolvedUpstream) string {
	// Budget by organization
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
func (g *Gateway) rateLimitedResult(rl RateLimitResult) PipelineResult {
	return PipelineResult{
		Allowed:      false,
		StatusCode:   429,
		ErrorType:    "rate_limit_exceeded",
		ErrorMessage: "rate limit exceeded",
		Context: &RequestContext{
			RateLimit: rl,
		},
	}
}

// budgetExceededResult creates a budget exceeded response.
func (g *Gateway) budgetExceededResult(br BudgetResult) PipelineResult {
	return PipelineResult{
		Allowed:      false,
		StatusCode:   402,
		ErrorType:    "budget_exceeded",
		ErrorMessage: "budget exceeded",
		Context: &RequestContext{
			BudgetResult: br,
		},
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

// writeResult writes the pipeline result to the response.
func (g *Gateway) writeResult(w http.ResponseWriter, result PipelineResult) {
	if result.Allowed {
		// Stream proxy will handle the actual forwarding
		// The caller should call StreamProxy.Forward with the context
		w.WriteHeader(result.StatusCode)
		return
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
func (g *Gateway) emitTelemetry(ctx context.Context, result PipelineResult, latencyMs int64) {
	if g.telemetry == nil {
		return
	}

	if result.Allowed && result.Context != nil {
		// Usage telemetry (sampled or all)
		usage := UsageRecord{
			OrganizationID:  result.Context.Credential.OrganizationID,
			CredentialID:    result.Context.Credential.ID,
			SubjectType:     result.Context.Credential.SubjectType,
			SubjectID:       result.Context.Credential.SubjectID,
			ProjectID:       result.Context.Credential.ProjectID,
			TeamID:          result.Context.Credential.TeamID,
			Model:           result.Context.InspectorHints.Model,
			InputTokens:     result.Context.InspectorHints.EstimatedInputTokens,
			OutputTokens:    result.Context.InspectorHints.EstimatedOutputTokens,
			LatencyMs:       latencyMs,
			StatusCode:      result.StatusCode,
			Timestamp:       time.Now(),
		}
		g.telemetry.EmitUsage(ctx, usage)
	} else {
		// Error telemetry
		errRecord := ErrorRecord{
			OrganizationID: uuid.Nil, // Would be set from context
			CredentialID:   uuid.Nil,
			ErrorType:      result.ErrorType,
			ErrorCode:      result.ErrorType,
			StatusCode:     result.StatusCode,
			LatencyMs:      latencyMs,
			Timestamp:      time.Now(),
			UpstreamError:  result.ErrorMessage,
		}
		g.telemetry.EmitError(ctx, errRecord)
	}
}

// Stats returns gateway statistics.
func (g *Gateway) Stats() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]any{
		"uptime_seconds":     time.Since(g.startTime).Seconds(),
		"request_count":      atomic.LoadInt64(&g.requestCount),
		"error_count":        atomic.LoadInt64(&g.errorCount),
		"active_streams":     g.streamProxy.(*StreamProxyImpl).ActiveStreamCount(),
		"started":            g.started,
		"start_time":         g.startTime,
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