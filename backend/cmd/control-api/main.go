// The control-api runs the governance control plane.
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

	if err := bootstrap.RunControlAPI(ctx); err != nil {
		slog.Error("control-api failed", "error", err)
		os.Exit(1)
	}
}
