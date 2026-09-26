// The migrate tool applies all pending database migrations to the
// control-plane Postgres named by AIGW_DATABASE_URL. It is idempotent and
// safe to run during deploys and local development alike.
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

	if err := bootstrap.RunMigrate(ctx); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}
