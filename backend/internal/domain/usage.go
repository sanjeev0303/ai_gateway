package domain

import (
	"time"

	"github.com/google/uuid"
)

// LlmRequest statuses, mirroring the CHECK in llm_requests.
const (
	LlmRequestStatusSucceeded = "succeeded"
	LlmRequestStatusFailed    = "failed"
	LlmRequestStatusCancelled = "cancelled"
)

// Usage event types and units, mirroring the CHECK in usage_events.
const (
	UsageEventTypeLLMRequest = "llm_request"
	UsageEventTypeTokenUsage = "token_usage"
	UsageEventTypeCost       = "cost"
)

const (
	UsageUnitCount  = "count"
	UsageUnitTokens = "tokens"
	UsageUnitMicros = "micros"
)

// Conversation statuses and message roles, mirroring their CHECKs.
const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)

const (
	MessageRoleSystem    = "system"
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleTool      = "tool"
)

// LlmRequest is a normalized record of one gateway request to a provider.
type LlmRequest struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	ProjectID           *uuid.UUID
	UserID              *uuid.UUID
	ServiceAccountID    *uuid.UUID
	TeamID              *uuid.UUID
	GatewayCredentialID *uuid.UUID
	RequestID           string
	ProviderRequestID   *string
	ProviderKind        string
	Model               string
	Status              string
	ErrorCode           *string
	PromptTokens        int32
	CompletionTokens    int32
	CachedInputTokens   int32
	TotalTokens         int32
	LatencyMs           int32
	CostMicros          int64
	CreatedAt           time.Time
	RequestStartedAt    *time.Time
	FirstTokenAt        *time.Time
	CompletedAt         *time.Time
	TimeToFirstTokenMs  *int32
	TotalLatencyMs      *int32
}

// UsageEvent is a granular, append-only accounting record.
type UsageEvent struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	RequestID           string
	EventType           string
	Quantity            int64
	Unit                string
	Metadata            []byte
	UserID              *uuid.UUID
	ServiceAccountID    *uuid.UUID
	TeamID              *uuid.UUID
	ProjectID           *uuid.UUID
	GatewayCredentialID *uuid.UUID
	CreatedAt           time.Time
}

// UsageRollup aggregates usage per (organization, project, model) bucket.
type UsageRollup struct {
	OrganizationID    uuid.UUID
	ProjectID         uuid.UUID
	Model             string
	BucketStart       time.Time
	Requests          int32
	PromptTokens      int64
	CompletionTokens  int64
	CachedInputTokens int64
	TotalTokens       int64
	CostMicros        int64
	UpdatedAt         time.Time
}

// UsageSummary is the aggregated output of a usage period. AvgLatencyMs is
// nil when the window held no requests, and LastActivityAt is the zero time
// in that same case, so callers never read a fabricated zero millisecond
// figure as a real measurement.
type UsageSummary struct {
	Requests        int64
	InputTokens     int64
	OutputTokens    int64
	CachedTokens    int64
	TotalTokens     int64
	TotalCostMicros int64
	Errors          int64
	AvgLatencyMs    *float64
	LastActivityAt  time.Time
}

// UsageTimeSeriesPoint is one point in a usage graph.
type UsageTimeSeriesPoint struct {
	BucketStart time.Time
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

// UsageFilter narrows a usage query. Nil members are ignored, so it composes
// with any subset of the entity and dimension filters.
type UsageFilter struct {
	From                time.Time
	To                  time.Time
	ProjectID           *uuid.UUID
	UserID              *uuid.UUID
	ServiceAccountID    *uuid.UUID
	TeamID              *uuid.UUID
	GatewayCredentialID *uuid.UUID
	Provider            *string
	Model               *string
	Status              *string
}

// UsageTotals is the metric set shared by every entity breakdown. It is
// embedded so it flattens onto the wire beside the entity's identity fields.
// AvgLatencyMs is nil when the group recorded no latency sample.
type UsageTotals struct {
	Requests       int64
	InputTokens    int64
	OutputTokens   int64
	TotalTokens    int64
	CostMicros     int64
	Errors         int64
	AvgLatencyMs   *float64
	LastActivityAt time.Time
}

// UsageByProject aggregates usage across projects.
type UsageByProject struct {
	ProjectID uuid.UUID
	Name      string
	UsageTotals
}

// UsageByModel aggregates usage by model.
type UsageByModel struct {
	Model       string
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

type UsageByProvider struct {
	Provider    string
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

// UsageByUser aggregates usage per user, with identity resolved by join.
type UsageByUser struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	UsageTotals
}

// UsageByServiceAccount aggregates usage per service account.
type UsageByServiceAccount struct {
	ServiceAccountID uuid.UUID
	Name             string
	UsageTotals
}

// UsageByTeam aggregates usage per team. Membership is many-to-many, so a
// request counts toward every team its user belongs to and team totals are not
// additive.
type UsageByTeam struct {
	TeamID uuid.UUID
	Name   string
	UsageTotals
}

// UsageLatency is a percentile summary of request timings. Percentiles are
// computed in Postgres so clients never aggregate raw samples. TTFT and total
// latency are counted separately because a non-streaming request has no
// time-to-first-token sample, so each family is nil only when it has none.
type UsageLatency struct {
	Provider     *string
	Model        *string
	TtftSamples  int64
	TotalSamples int64
	AvgTTFTMs    *float64
	P50TTFTMs    *float64
	P95TTFTMs    *float64
	P99TTFTMs    *float64
	AvgTotalMs   *float64
	P50TotalMs   *float64
	P95TotalMs   *float64
	P99TotalMs   *float64
}

// NewLlmRequest carries the caller-supplied fields for recording one gateway
// request. Status is the recorded outcome, so it is always supplied.
type NewLlmRequest struct {
	ProjectID           *uuid.UUID
	UserID              *uuid.UUID
	ServiceAccountID    *uuid.UUID
	TeamID              *uuid.UUID
	GatewayCredentialID *uuid.UUID
	RequestID           string
	ProviderRequestID   *string
	ProviderKind        string
	Model               string
	Status              string
	ErrorCode           *string
	PromptTokens        int32
	CompletionTokens    int32
	CachedInputTokens   int32
	TotalTokens         int32
	LatencyMs           int32
	CostMicros          int64
	CreatedAt           time.Time
	RequestStartedAt    *time.Time
	FirstTokenAt        *time.Time
	CompletedAt         *time.Time
	TimeToFirstTokenMs  *int32
	TotalLatencyMs      *int32
}

// NewUsageEvent carries the caller-supplied fields for recording a usage
// event. Metadata defaults to '{}' in the database when nil.
type NewUsageEvent struct {
	RequestID           string
	EventType           string
	Quantity            int64
	Unit                string
	Metadata            []byte
	UserID              *uuid.UUID
	ServiceAccountID    *uuid.UUID
	TeamID              *uuid.UUID
	ProjectID           *uuid.UUID
	GatewayCredentialID *uuid.UUID
}

// IdempotencyKey is a request-scoped deduplication key.
type IdempotencyKey struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ScopeKey       string
	Key            string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// NewIdempotencyKey carries the caller-supplied fields for registering an
// idempotency key. ExpiresAt is required.
type NewIdempotencyKey struct {
	ScopeKey  string
	Key       string
	ExpiresAt time.Time
}
