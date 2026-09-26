// The worker runs async jobs: usage rollups, retention, alerts.
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
