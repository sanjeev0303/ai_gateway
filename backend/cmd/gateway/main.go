// The gateway runs the data plane hot path.
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

	if err := bootstrap.RunGateway(ctx); err != nil {
		slog.Error("gateway failed", "error", err)
		os.Exit(1)
	}
}
