package worker

import (
	"context"
	"log/slog"

	"ai-governance-saas/backend/internal/application/usage"
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
