// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	// Policy
	PolicyRepo repositories.PolicyRepository
	RuleRepo   repositories.PolicyRuleRepository

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
		budgetGuard = NewBudgetGuard()
	}

	// Telemetry
	telemetry := deps.Telemetry
	if telemetry == nil {
		telemetry = NewTelemetryEmitter(10000, 5*time.Second, deps.EventBus, deps.Logger)
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
		
		observed, err := g.streamProxy.Forward(r.Context(), w, r, result.Context.UpstreamURL, result.Context.UpstreamHeaders)
		if err != nil {
			g.logger.Error("stream proxy forward failed", "error", err)
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

	// Stage 4: Inspect request for routing hints. The body must survive
	// inspection, so it is buffered once under a bound and restored.
	body, err := readAndRestoreBody(r)
	if err != nil {
		return g.pipelineError("request_inspector", err, http.StatusRequestEntityTooLarge, "request_too_large")
	}

	hints, err := g.requestInspector.Inspect(ctx, bytes.NewReader(body))
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

// writeResult writes a denied pipeline result to the response. Allowed results
// never reach here: ServeHTTP forwards them upstream and writes the response
// from the provider, so this path is deny-only by design.
func (g *Gateway) writeResult(w http.ResponseWriter, result PipelineResult) {
	if result.Allowed {
		g.logger.Error("writeResult called for an allowed result; upstream response was not written")
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
			if result.StatusCode >= 400 {
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
		usage := UsageRecord{
			RequestID:        reqID,
			OrganizationID:   result.Context.Credential.OrganizationID,
			CredentialID:     result.Context.Credential.ID,
			SubjectType:      result.Context.Credential.SubjectType,
			SubjectID:        result.Context.Credential.SubjectID,
			ProjectID:        result.Context.Credential.ProjectID,
			TeamID:           result.Context.Credential.TeamID,
			Model:            result.Context.InspectorHints.Model,
			InputTokens:      result.Context.InspectorHints.EstimatedInputTokens,
			OutputTokens:     result.Context.InspectorHints.EstimatedOutputTokens,
			LatencyMs:        latencyMs,
			StatusCode:       result.StatusCode,
			Timestamp:        completedAt,
			RequestStartedAt: startUTC,
			CompletedAt:      completedAt,
		}
		
		totalLatencyMs := int32(latencyMs)
		usage.TotalLatencyMs = &totalLatencyMs

		// Provider-reported counts are authoritative; the inspector only
		// estimates, so it is the fallback for responses carrying no usage.
		if observed != nil {
			usage.InputTokens = observed.InputTokens
			usage.OutputTokens = observed.OutputTokens
			usage.CacheReadTokens = observed.CacheReadTokens
			usage.CacheWriteTokens = observed.CacheWriteTokens
			usage.ProviderMeta = map[string]any{"provider": observed.Provider}
			
			if !observed.FirstTokenAt.IsZero() {
				firstToken := observed.FirstTokenAt.UTC()
				usage.FirstTokenAt = &firstToken
				ttft := int32(firstToken.Sub(startUTC).Milliseconds())
				usage.TimeToFirstTokenMs = &ttft
			}
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
