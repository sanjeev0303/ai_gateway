package worker

import (
	"context"
	"log/slog"

	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/domain"
	"encoding/json"
	"ai-governance-saas/backend/internal/ports"
)

// RunUsageConsumer subscribes to the usage event bus and hands every message to
// the usage service, which owns the durable ledger writes. The worker is the
// only process that persists usage, so a redelivered message is absorbed by the
// ledger's idempotency rather than double-counted.
func RunUsageConsumer(ctx context.Context, bus ports.EventBus, svc *usage.Service, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	if err := bus.Run(ctx, func(ctx context.Context, msg ports.UsageEventMessage) error {
		return svc.RecordRequest(ctx, msg.OrganizationID, msg.Request, msg.Usage)
	}); err != nil && ctx.Err() == nil {
		log.Error("usage consumer stopped", "error", err)
		return
	}
	log.Info("usage consumer stopped")
}

// RunSecurityConsumer subscribes to the security event bus and hands every message
// to the security event service to persist.
func RunSecurityConsumer(ctx context.Context, bus ports.EventBus, svc *securityevent.Service, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	if err := bus.RunSecurity(ctx, func(ctx context.Context, msg ports.SecurityEventMessage) error {
		detail, _ := json.Marshal(msg.Details)
		return svc.Record(ctx, msg.OrganizationID, domain.NewSecurityEvent{
			Category: domain.SecurityEventCategory(msg.Category),
			Severity: domain.SecurityEventSeverity(msg.Severity),
			Title:    msg.Title,
			Source:   msg.Source,
			Detail:   detail,
		})
	}); err != nil && ctx.Err() == nil {
		log.Error("security consumer stopped", "error", err)
		return
	}
	log.Info("security consumer stopped")
}
