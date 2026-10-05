package repositories

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestUniqueViolationOn(t *testing.T) {
	email := &pgconn.PgError{Code: "23505", ConstraintName: "users_email_lower_key"}
	slug := &pgconn.PgError{Code: "23505", ConstraintName: "organizations_slug_key"}

	if !uniqueViolationOn(email, "users_email_lower_key") {
		t.Error("email unique violation not detected")
	}
	if uniqueViolationOn(slug, "users_email_lower_key") {
		t.Error("slug unique violation matched the email constraint")
	}
	if uniqueViolationOn(errors.New("boom"), "users_email_lower_key") {
		t.Error("plain error matched a constraint")
	}
}
