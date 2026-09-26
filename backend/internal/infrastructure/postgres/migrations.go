package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending goose migrations, creating the version table
// if necessary. It is idempotent: already-applied migrations are skipped.
// The schema (RLS in 00012, membership lookup in 00013) is built entirely
// here, so application code can assume it exists once Migrate returns nil.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// The embedded FS nests migrations under migrations/, but goose expects
	// the SQL files at the FS root, so expose a sub-filesystem rooted there.
	fsys, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	// Connections created before Up (the eager ping in NewStore, which in
	// tests runs before migrations) skipped type registration in AfterConnect
	// because the types did not exist yet. Reset rebuilds the pool so every
	// connection re-runs the hook with the enum types now present.
	pool.Reset()
	return nil
}
