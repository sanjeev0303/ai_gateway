package activity_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/activity"
)

func TestService_NilClientReturnsEmptySlice(t *testing.T) {
	svc := activity.NewService(nil, nil)
	orgID := uuid.New()

	events, err := svc.ActiveRequests(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if events == nil {
		t.Fatal("expected empty non-nil slice, got nil")
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	ch, err := svc.Subscribe(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Channel should be immediately closed
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel for nil client")
	}
}
