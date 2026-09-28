package worker

import (
	"context"
	"log/slog"
	"time"

	"ai-governance-saas/backend/internal/application/usage"
)

const (
	// rollupInterval is how often rollups are rebuilt.
	rollupInterval = 5 * time.Minute

	// rollupLookback is how far back each rebuild reaches. It is far wider
	// than rollupInterval on purpose: the loop keeps no watermark, so this
	// window is the only record of how long the job was absent, and reaching
	// past it is what lets a restart or a missed tick repair the gap. Rebuilds
	// are idempotent, so the redundant overlap costs scans, not correctness.
	rollupLookback = 24 * time.Hour
)

func RunRollup(ctx context.Context, svc *usage.Service, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	ticker := time.NewTicker(rollupInterval)
	defer ticker.Stop()

	for {
		to := time.Now().UTC()
		if err := svc.RecomputeAll(ctx, to.Add(-rollupLookback), to); err != nil {
			log.Error("usage rollup pass failed", "error", err)
		}

		select {
		case <-ctx.Done():
			log.Info("usage rollup loop stopped")
			return
		case <-ticker.C:
		}
	}
}
