package ports

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// UsageEventMessage carries one completed provider request from the data plane
// to the worker that owns the durable ledger.
type UsageEventMessage struct {
	OrganizationID uuid.UUID
	Request        domain.NewLlmRequest
	Usage          domain.NewUsageEvent
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
