package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed seed.sql
var seedSQL string

// Seed applies the idempotent demo seed data from seed.sql. It assumes the
// schema exists: run Migrate first (cmd/seed does both). Re-running Seed on
// an already-seeded database is a no-op thanks to the WHERE NOT EXISTS guards
// in seed.sql.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if _, err := db.ExecContext(ctx, seedSQL); err != nil {
		return fmt.Errorf("apply seed: %w", err)
	}
	return nil
}
