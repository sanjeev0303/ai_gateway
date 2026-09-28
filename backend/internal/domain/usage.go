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
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	ProjectID         *uuid.UUID
	UserID            *uuid.UUID
	APICredentialID   *uuid.UUID
	RequestID         string
	ProviderRequestID *string
	ProviderKind      string
	Model             string
	Status            string
	ErrorCode         *string
	PromptTokens      int32
	CompletionTokens  int32
	CachedInputTokens int32
	TotalTokens       int32
	LatencyMs         int32
	CostMicros        int64
	CreatedAt         time.Time
	RequestStartedAt  *time.Time
	FirstTokenAt      *time.Time
	CompletedAt       *time.Time
	TimeToFirstTokenMs *int32
	TotalLatencyMs    *int32
}

// UsageEvent is a granular, append-only accounting record.
type UsageEvent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RequestID      string
	EventType      string
	Quantity       int64
	Unit           string
	Metadata       []byte
	CreatedAt      time.Time
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

// UsageSummary is the aggregated output of a usage period.
type UsageSummary struct {
	Requests       int64
	InputTokens    int64
	OutputTokens   int64
	CachedTokens   int64
	TotalTokens    int64
	TotalCostMicros int64
}

// UsageTimeSeriesPoint is one point in a usage graph.
type UsageTimeSeriesPoint struct {
	BucketStart time.Time
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

// UsageByProject aggregates usage across projects.
type UsageByProject struct {
	ProjectID   uuid.UUID
	Requests    int64
	TotalTokens int64
	CostMicros  int64
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

type UsageByUser struct {
	UserID      uuid.UUID
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

type UsageByTeam struct {
	TeamID      uuid.UUID
	Requests    int64
	TotalTokens int64
	CostMicros  int64
}

// Conversation is a user's persisted chat session.
type Conversation struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Title          string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ConversationMessage is one turn within a conversation.
type ConversationMessage struct {
	ID                uuid.UUID
	ConversationID    uuid.UUID
	OrganizationID    uuid.UUID
	UserID            uuid.UUID
	Role              string
	Content           string
	Tokens            int32
	ProviderMessageID *string
	CreatedAt         time.Time
}

// NewConversation carries the caller-supplied fields for creating a
// conversation. Status is defaulted by the database to 'active'.
type NewConversation struct {
	UserID uuid.UUID
	Title  string
}

// NewConversationMessage carries the caller-supplied fields for appending a
// message to a conversation.
type NewConversationMessage struct {
	ConversationID    uuid.UUID
	UserID            uuid.UUID
	Role              string
	Content           string
	Tokens            int32
	ProviderMessageID *string
}

// NewLlmRequest carries the caller-supplied fields for recording one gateway
// request. Status is the recorded outcome, so it is always supplied.
type NewLlmRequest struct {
	ProjectID         *uuid.UUID
	UserID            *uuid.UUID
	APICredentialID   *uuid.UUID
	RequestID         string
	ProviderRequestID *string
	ProviderKind      string
	Model             string
	Status            string
	ErrorCode         *string
	PromptTokens      int32
	CompletionTokens  int32
	CachedInputTokens int32
	TotalTokens       int32
	LatencyMs         int32
	CostMicros        int64
	CreatedAt         time.Time
	RequestStartedAt  *time.Time
	FirstTokenAt      *time.Time
	CompletedAt       *time.Time
	TimeToFirstTokenMs *int32
	TotalLatencyMs    *int32
}

// NewUsageEvent carries the caller-supplied fields for recording a usage
// event. Metadata defaults to '{}' in the database when nil.
type NewUsageEvent struct {
	RequestID string
	EventType string
	Quantity  int64
	Unit      string
	Metadata  []byte
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
