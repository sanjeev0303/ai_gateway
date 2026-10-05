package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	ports "ai-governance-saas/backend/internal/ports/repositories"
)

// A zero time must become SQL NULL, not year 1, or an omitted date filter
// would silently match everything.
func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// mapNotFound converts pgx.ErrNoRows from Get-style queries into the
// port-level ErrNotFound sentinel.
func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %v", ports.ErrNotFound, err)
	}
	return err
}

// mapConflict converts pgx.ErrNoRows from idempotent inserts (ON CONFLICT DO
// NOTHING RETURNING) into the port-level ErrConflict sentinel.
func mapConflict(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %v", ports.ErrConflict, err)
	}
	return err
}

// mapUniqueViolation converts Postgres ERROR 23505 (unique_violation) into the
// port-level ErrConflict sentinel.
func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %v", ports.ErrConflict, err)
	}
	return err
}

// uniqueViolationOn reports whether err is a Postgres 23505 (unique_violation)
// raised by the named constraint or unique index. Registration uses it to tell
// an org-slug collision (retryable) from an account-email collision (not).
func uniqueViolationOn(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// nonEmptyStringPtr returns a pointer to s when s is non-empty, else nil.
// Used for nullable columns whose NULL carries meaning (e.g. legacy rows
// predating a fingerprint column).
func nonEmptyStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
