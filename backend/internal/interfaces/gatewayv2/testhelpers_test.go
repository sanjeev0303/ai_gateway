// Package gatewayv2 contract tests (in-package: unexported seams like clocks,
// caches and defaults stay reachable). Convention matches the rest of the
// module: hand-written stubs, stdlib testing, no testify/gomock.
package gatewayv2

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

var mustUUIDs = []uuid.UUID{
	uuid.MustParse("11111111-1111-4111-8111-111111111111"),
	uuid.MustParse("22222222-2222-4222-8222-222222222222"),
	uuid.MustParse("33333333-3333-4333-8333-333333333333"),
}

// requireNoErr fails the test when err is non-nil. msg is optional context.
func requireNoErr(t *testing.T, err error, msg ...string) {
	t.Helper()
	if err != nil {
		if len(msg) > 0 {
			t.Fatalf("%s: unexpected error: %v", msg[0], err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireErrIs fails the test unless err wraps target.
func requireErrIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expected error %v, got %v", target, err)
	}
}

// requireTrue fails the test when cond is false. format may contain verbs.
func requireTrue(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

// discardLogger returns a logger that drops every record; use it anywhere a
// component requires *slog.Logger but the test does not assert on logs.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// requireEqual fails the test when got != want.
func requireEqual[T comparable](t *testing.T, got, want T, msg ...string) {
	t.Helper()
	if got != want {
		if len(msg) > 0 {
			t.Fatalf("%s: got %v, want %v", msg[0], got, want)
		}
		t.Fatalf("got %v, want %v", got, want)
	}
}

// strPtr returns a pointer to s for building pointer-typed test fixtures.
func strPtr(s string) *string {
	return &s
}
