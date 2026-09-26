package ids

import (
	"strings"
	"testing"
)

func TestNewFormat(t *testing.T) {
	id := New()
	if len(id) != 32 {
		t.Errorf("New() length = %d, want 32", len(id))
	}
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("New() returned non-hex char %q", r)
		}
	}
}

func TestNewUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		id := New()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestNewCorrelationID(t *testing.T) {
	a, b := NewCorrelationID(), NewCorrelationID()
	if a == b {
		t.Error("two correlation IDs collided")
	}
	if len(a) != 32 {
		t.Errorf("correlation id length = %d, want 32", len(a))
	}
}
