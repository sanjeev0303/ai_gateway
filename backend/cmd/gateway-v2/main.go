// The gateway-v2 runs the latency-sensitive data plane hot path.
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

	if err := bootstrap.RunGatewayV2(ctx); err != nil {
		slog.Error("gateway-v2 failed", "error", err)
		os.Exit(1)
	}
}
