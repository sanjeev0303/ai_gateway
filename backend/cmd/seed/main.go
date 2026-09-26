// The seed tool applies all pending database migrations and then the
// idempotent demo seed data, leaving a fresh control-plane database fully
// usable for local development. Re-running it is safe (no-op on seeded data).
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

	if err := bootstrap.RunSeed(ctx); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}
