package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/platform/config"
	"ai-governance-saas/backend/internal/platform/logging"
)

// RunMigrate applies all pending goose migrations to the control-plane
// database named by AIGW_DATABASE_URL. It is safe to run any number of times.
func RunMigrate(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if cfg.Database.URL == "" {
		return errors.New("migrate: AIGW_DATABASE_URL is required")
	}
	log := logging.New(cfg.Observability)
	log.Info("applying database migrations", "process", "migrate")

	store, err := postgres.NewStore(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	defer store.Close()

	if err := postgres.Migrate(ctx, store.Pool()); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("database migrations applied")
	return nil
}

// RunSeed applies pending migrations and then the idempotent demo seed data,
// so a fresh database is fully usable after `go run ./cmd/seed`. Re-running
// it on an already-seeded database is a no-op.
func RunSeed(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	if cfg.Database.URL == "" {
		return errors.New("seed: AIGW_DATABASE_URL is required")
	}
	log := logging.New(cfg.Observability)
	log.Info("seeding database", "process", "seed")

	store, err := postgres.NewStore(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	defer store.Close()

	if err := postgres.Migrate(ctx, store.Pool()); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	if err := postgres.Seed(ctx, store.Pool()); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	log.Info("database seeded")
	return nil
}
