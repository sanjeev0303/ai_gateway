package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ErrNonRetryable indicates a failure that will never succeed on redelivery
// (e.g., malformed payload or permanent validation failure).
var ErrNonRetryable = errors.New("eventbus: non-retryable error")

// UsageEventMessage carries one completed provider request from the data plane
// to the worker that owns the durable ledger.
type UsageEventMessage struct {
	OrganizationID uuid.UUID
	Request        domain.NewLlmRequest
	Usage          domain.NewUsageEvent
}

// DeadLetterUsageMessage is routed to the dead-letter queue when a usage event
// permanently fails or exhausts delivery attempts.
type DeadLetterUsageMessage struct {
	OriginalMessage UsageEventMessage `json:"original_message"`
	Error           string            `json:"error"`
	Deliveries      uint64            `json:"deliveries"`
	FailedAt        time.Time         `json:"failed_at"`
}

// SecurityEventMessage carries a security-relevant event from the data plane.
type SecurityEventMessage struct {
	OrganizationID uuid.UUID
	Category       string
	Severity       string
	Title          string
	Source         string
	Details        map[string]any
}

// EventBus decouples the gateway data plane from the worker's usage ledger.
type EventBus interface {
	Publish(ctx context.Context, msg UsageEventMessage) error
	PublishSecurityEvent(ctx context.Context, msg SecurityEventMessage) error
	Run(ctx context.Context, handle func(context.Context, UsageEventMessage) error) error
	RunSecurity(ctx context.Context, handle func(context.Context, SecurityEventMessage) error) error
}

