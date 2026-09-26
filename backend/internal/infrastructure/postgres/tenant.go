package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
)

// WithTenantTx runs fn inside a transaction that first binds the tenant
// context to the session: set_config('app.org_id', ..., true) has SET LOCAL
// semantics, so the value lives for the transaction only. Every tenant table
// is FORCE RLS (migration 00012) and its policy compares organization_id to
// app.current_org_id(), which reads that GUC; a forgotten bind fails loudly
// (no rows) instead of leaking across tenants.
func (s *Store) WithTenantTx(ctx context.Context, orgID uuid.UUID, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.org_id', $1, true)", orgID.String()); err != nil {
		return fmt.Errorf("bind tenant context: %w", err)
	}

	if err := fn(db.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// WithTx runs fn inside a plain transaction with no tenant binding. Use it
// only for global tables (organizations, users, plans), which are not RLS
// protected. Cross-tenant lookups (API credential by key hash, organizations
// for a user) run inside SECURITY DEFINER functions defined in the
// migrations and need no tenant context either.
func (s *Store) WithTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if err := fn(db.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
