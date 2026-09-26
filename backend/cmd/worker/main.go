// The worker currently serves health probes only. Async jobs (usage rollups,
// retention, alerts) are not wired up yet.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ai-governance-saas/backend/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := bootstrap.RunWorker(ctx); err != nil {
		slog.Error("worker failed", "error", err)
		os.Exit(1)
	}
}
