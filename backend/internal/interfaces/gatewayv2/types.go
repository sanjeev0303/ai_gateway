// Package gatewayv2 implements the latency-sensitive data-plane gateway.
// It provides a modular, high-performance proxy for LLM provider traffic.
package gatewayv2

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ---- Core Types -------------------------------------------------------------

// GatewayCredential represents a resolved gateway credential.
type GatewayCredential struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	SubjectType       string // "user" | "service_account"
	SubjectID         uuid.UUID
	TeamID            *uuid.UUID
	ProjectID         *uuid.UUID
	Scopes            []string
	ProviderAccountID *uuid.UUID
	AllowedProviders  []string
	AllowedModels     []string
	RateLimit         *int
	BudgetLimit       *float64
	Status            string
	ExpiresAt         *time.Time
}

// ProviderAccount represents an upstream provider connection.
type ProviderAccount struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProviderKind   string
	Name           string
	BaseURL        string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ProviderCredential represents an upstream credential.
type ProviderCredential struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	ProviderAccountID uuid.UUID
	SecretRef         string
	Fingerprint       string
	Tail              string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ResolvedUpstream contains the upstream connection details.
type ResolvedUpstream struct {
	ProviderAccount ProviderAccount
	Credential      ProviderCredential
	UpstreamKey     string
	BaseURL         string
}

// RequestContext carries the fully resolved request context through the pipeline.
type RequestContext struct {
	Credential      GatewayCredential
	Upstream        ResolvedUpstream
	PolicySnapshot  PolicySnapshot
	RateLimit       RateLimitResult
	BudgetResult    BudgetResult
	BudgetRelease   BudgetReleaseFunc
	InspectorHints  InspectorHints
	StartTime       time.Time
	UpstreamURL     string
	UpstreamHeaders http.Header
}

// PolicySnapshot is an immutable policy view for a single request.
type PolicySnapshot struct {
	OrganizationID uuid.UUID
	Version        int64
	Rules          []PolicyRule
	RateLimits     map[string]RateLimitConfig
	Budgets        map[string]BudgetConfig
	EnforcedAt     time.Time
}

type PolicyRule struct {
	ID         string
	PolicyID   string
	Priority   int32
	Type       string // domain rule type: rate_limit | token_budget | blocklist | allowlist | content_filter | budget
	Scope      string // "organization" | "team" | "project" | "user" | "service_account" | "credential"
	ScopeID    string
	Conditions map[string]any
	Actions    map[string]any
	Enabled    bool
}

type RateLimitConfig struct {
	Key        string
	Limit      int
	Window     time.Duration
	Scope      string // "organization" | "team" | "project" | "credential" | "user" | "service_account"
	ScopeID    string
	BurstAllow int
}

type BudgetConfig struct {
	Key       string
	Limit     float64
	Window    time.Duration
	Scope     string
	ScopeID   string
	LimitType string // "spend" | "tokens" | "requests"
	AlertAt   float64
}

// RateLimitResult captures rate limiting decision.
type RateLimitResult struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
	Scope      string
	Key        string
	Degraded   bool
}

// BudgetResult captures budget checking decision.
type BudgetResult struct {
	Allowed   bool
	Current   float64
	Limit     float64
	Projected float64
	AlertAt   float64
	Scope     string
	Key       string
	Degraded  bool
}

// InspectorHints carries request inspection results for routing.
type InspectorHints struct {
	Model                 string
	Provider              string
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	Tools                 []string
	Stream                bool
	ConversationID        *uuid.UUID
	ProjectID             *uuid.UUID
	RoutingTags           map[string]string
	Warnings              []string
}

// UpstreamResponse represents the upstream provider response.
type UpstreamResponse struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
	Trailers   http.Header
}

// ---- Component Interfaces ---------------------------------------------------

// Authenticator validates credentials and returns the gateway credential.
type Authenticator interface {
	Authenticate(ctx context.Context, rawKey string) (GatewayCredential, error)
}

// TenantResolver resolves the upstream provider connection for an organization.
type TenantResolver interface {
	ResolveUpstream(ctx context.Context, orgID uuid.UUID) (ResolvedUpstream, error)
	ResolveUpstreamForProvider(ctx context.Context, orgID uuid.UUID, providerKind string) (ResolvedUpstream, error)
	ResolveUpstreamForAccount(ctx context.Context, orgID, accountID uuid.UUID) (ResolvedUpstream, error)
	ListActiveProviders(ctx context.Context, orgID uuid.UUID) ([]string, error)
	Invalidate(orgID uuid.UUID)
}

// PolicyResolver provides immutable policy snapshots for evaluation.
type PolicyResolver interface {
	GetSnapshot(ctx context.Context, orgID uuid.UUID) (PolicySnapshot, error)
	Invalidate(orgID uuid.UUID)
}

// RateLimiter enforces rate limits.
type RateLimiter interface {
	Check(ctx context.Context, key string, limit int, window time.Duration) RateLimitResult
}

// BudgetReleaseFunc finalizes or releases a budget reservation.
// actualCost is the measured spend (0 for failures or cancellations).
// err indicates if the request failed or was interrupted.
type BudgetReleaseFunc func(actualCost float64, err error)

// BudgetGuard enforces budget limits and manages reservation lifecycles.
type BudgetGuard interface {
	Check(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) BudgetResult
	Reserve(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, reqID string) (BudgetResult, BudgetReleaseFunc)
	RecordUsage(ctx context.Context, key string, cost float64) error
}

// RequestInspector validates and extracts routing hints from requests.
type RequestInspector interface {
	Inspect(ctx context.Context, body io.Reader) (InspectorHints, error)
}

// ProviderRouter selects and configures the upstream provider connection.
type ProviderRouter interface {
	Route(ctx context.Context, upstream ResolvedUpstream, hints InspectorHints) (string, http.Header, error)
}

// StreamProxy forwards requests and streams responses byte-exact. Forward
// returns the provider-reported usage observed on the response, or nil when
// the upstream sent none. Usage is returned even alongside an error, because a
// client that disconnects after the final usage frame has still been billed.
type StreamProxy interface {
	Forward(ctx context.Context, w http.ResponseWriter, r *http.Request, upstreamURL string, upstreamHeaders http.Header) (*StreamUsage, error)
	Close(gracePeriod time.Duration) error
}

// TelemetryEmitter emits usage and telemetry asynchronously.
type TelemetryEmitter interface {
	EmitUsage(ctx context.Context, usage UsageRecord)
	EmitError(ctx context.Context, err ErrorRecord)
	Flush(ctx context.Context) error
	Close() error
}

// UsageRecord represents a usage event for telemetry.
type UsageRecord struct {
	RequestID          string
	OrganizationID     uuid.UUID
	CredentialID       uuid.UUID
	SubjectType        string
	SubjectID          uuid.UUID
	ProjectID          *uuid.UUID
	TeamID             *uuid.UUID
	Model              string
	InputTokens        int
	OutputTokens       int
	CacheReadTokens    int
	CacheWriteTokens   int
	ReasoningTokens    int
	ToolTokens         int
	ToolCalls          int
	LatencyMs          int64
	StatusCode         int
	Timestamp          time.Time
	RequestStartedAt   time.Time
	FirstTokenAt       *time.Time
	CompletedAt        time.Time
	TimeToFirstTokenMs *int32
	TotalLatencyMs     *int32
	CostMicros         int64
	PricingVersion     *int32
	CostStatus         string
	PolicyDecision     *string
	DenialReason       *string
	ProviderMeta       map[string]any
}

// ErrorRecord represents an error event for telemetry.
type ErrorRecord struct {
	OrganizationID uuid.UUID
	CredentialID   uuid.UUID
	ErrorType      string
	ErrorCode      string
	StatusCode     int
	LatencyMs      int64
	Timestamp      time.Time
	UpstreamError  string
}

// ---- Pipeline Result Types --------------------------------------------------

// PipelineResult represents the outcome of the gateway pipeline.
type PipelineResult struct {
	Allowed      bool
	StatusCode   int
	ErrorType    string
	ErrorMessage string
	Context      *RequestContext
}

// PipelineError represents a pipeline error with context.
type PipelineError struct {
	Stage      string
	Err        error
	StatusCode int
	ErrorType  string
	Retryable  bool
}

func (e *PipelineError) Error() string {
	return e.Stage + ": " + e.Err.Error()
}

// ---- Gateway Configuration --------------------------------------------------

// GatewayConfig holds the gateway configuration.
type GatewayConfig struct {
	// Timeouts
	RequestTimeout        time.Duration
	UpstreamTimeout       time.Duration
	IdleTimeout           time.Duration
	ResponseHeaderTimeout time.Duration

	// Connection pooling
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int

	// Buffer sizes
	ReadBufferSize  int
	WriteBufferSize int

	// Queue bounds
	MaxQueuedRequests int

	// Feature flags
	EnableHTTP2       bool
	EnableCompression bool

	// Default limits
	DefaultRateLimit    int
	DefaultRateWindow   time.Duration
	DefaultBudgetLimit  float64
	DefaultBudgetWindow time.Duration
}

// DefaultGatewayConfig returns sensible defaults.
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		RequestTimeout:        30 * time.Second,
		UpstreamTimeout:       5 * time.Minute,
		IdleTimeout:           90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       200,
		ReadBufferSize:        32 * 1024,
		WriteBufferSize:       32 * 1024,
		MaxQueuedRequests:     10000,
		EnableHTTP2:           true,
		EnableCompression:     false, // byte-exact proxy
		DefaultRateLimit:      1000,
		DefaultRateWindow:     time.Minute,
		DefaultBudgetLimit:    1000.0,
		DefaultBudgetWindow:   time.Hour,
	}
}
