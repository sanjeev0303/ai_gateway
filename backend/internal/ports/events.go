package ports

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// UsageEventMessage carries one completed provider request from the data plane
// to the worker that owns the durable ledger. The gateway derives both records
// from the upstream response; the worker performs every write, so the data plane
// holds no analytics persistence responsibility.
type UsageEventMessage struct {
	OrganizationID uuid.UUID
	Request        domain.NewLlmRequest
	Usage          domain.NewUsageEvent
}

// EventBus decouples the gateway data plane from the worker's usage ledger. A
// successful client response must never wait on analytics persistence, so
// Publish is expected to be non-blocking or bounded.
type EventBus interface {
	Publish(ctx context.Context, msg UsageEventMessage) error
	Run(ctx context.Context, handle func(context.Context, UsageEventMessage) error) error
}
