package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/infrastructure/postgres"
)

func TestMigrate_FailsVisiblyOnUnreachableDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Intentionally unreachable PostgreSQL port
	badDSN := "postgres://gateway:gateway@127.0.0.1:54329/ai_gateway?sslmode=disable&connect_timeout=1"
	store, err := postgres.NewStore(ctx, badDSN)
	if err == nil {
		if store != nil {
			store.Close()
		}
		t.Fatal("expected NewStore to fail on unreachable database, but got nil error")
	}

	if !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "refused") && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected clear connection error message, got: %v", err)
	}
}
