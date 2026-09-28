package gatewayv2

import (
	"context"
	"testing"
	"time"
)

func testCacheEntry(id, orgID string) *CredentialCacheEntry {
	return &CredentialCacheEntry{
		ID:             id,
		OrganizationID: orgID,
		SubjectType:    "service_account",
		SubjectID:      id,
		Scopes:         []string{"gateway"},
		Status:         "active",
	}
}

func TestCache_SetGetRoundTrip(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	entry := testCacheEntry("key-hash-1", mustUUIDs[1].String())

	c.Set(context.Background(), entry)
	got := c.Get(context.Background(), "key-hash-1")
	if got != entry {
		t.Fatal("expected the same entry pointer back")
	}
}

func TestCache_GetUnknownKeyReturnsNil(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	if got := c.Get(context.Background(), "missing"); got != nil {
		t.Fatalf("expected nil for unknown key, got %v", got)
	}
}

func TestCache_SetNilEntryIsNoop(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.Set(context.Background(), nil)
	size, _ := c.Stats()
	if size != 0 {
		t.Fatalf("expected size 0 after nil set, got %d", size)
	}
}

func TestCache_NullEntriesAllocated(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	var nilPtr *CredentialCacheEntry
	c.Set(context.Background(), nilPtr)
	size, _ := c.Stats()
	if size != 0 {
		t.Fatalf("expected size 0, got %d", size)
	}
}

func TestCache_ExpiresStaleEntryOnGet(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.clock = func() time.Time { return now }

	c.Set(context.Background(), testCacheEntry("key-hash-1", mustUUIDs[1].String()))

	// Set stamps UpdatedAt with the clock; advance the clock past TTL.
	c.clock = func() time.Time { return now.Add(2 * time.Hour) }
	if got := c.Get(context.Background(), "key-hash-1"); got != nil {
		t.Fatal("expected stale entry to be removed on Get")
	}
	if size, _ := c.Stats(); size != 0 {
		t.Fatalf("expected stale entry removed from map, got size %d", size)
	}
}

func TestCache_RemovesExpiredByExpiresAt(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.clock = func() time.Time { return now }

	past := now.Add(-time.Minute)
	entry := testCacheEntry("key-hash-1", mustUUIDs[1].String())
	entry.ExpiresAt = &past

	c.Set(context.Background(), entry)
	if got := c.Get(context.Background(), "key-hash-1"); got != nil {
		t.Fatal("expected credential expired by ExpiresAt to be removed")
	}
}

func TestCache_KeepsFreshEntry(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.clock = func() time.Time { return now }

	entry := testCacheEntry("key-hash-1", mustUUIDs[1].String())
	c.Set(context.Background(), entry)
	if got := c.Get(context.Background(), "key-hash-1"); got != entry {
		t.Fatal("expected fresh entry to be returned")
	}
}

func TestCache_GetMovesToFront(t *testing.T) {
	c := NewInMemoryCredentialCache(2, time.Hour)
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("b", mustUUIDs[0].String()))
	c.Get(context.Background(), "a")

	// "a" was touched most recently, so "b" is now the LRU tail.
	c.Set(context.Background(), testCacheEntry("c", mustUUIDs[0].String()))

	if got := c.Get(context.Background(), "a"); got == nil {
		t.Fatal("expected a to survive; it was most recently used")
	}
	if got := c.Get(context.Background(), "b"); got != nil {
		t.Fatal("expected b (LRU before c) to be evicted")
	}
	if got := c.Get(context.Background(), "c"); got == nil {
		t.Fatal("expected c to be present")
	}
}

func TestCache_EvictsLRUWhenFull(t *testing.T) {
	c := NewInMemoryCredentialCache(2, time.Hour)
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("b", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("c", mustUUIDs[0].String()))

	if got := c.Get(context.Background(), "a"); got != nil {
		t.Fatal("expected a to be evicted as LRU")
	}
	size, _ := c.Stats()
	if size != 2 {
		t.Fatalf("expected size 2 after eviction, got %d", size)
	}
}

func TestCache_UpdateExistingDoesNotDuplicate(t *testing.T) {
	c := NewInMemoryCredentialCache(2, time.Hour)
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[1].String()))

	size, _ := c.Stats()
	if size != 1 {
		t.Fatalf("expected size 1 after updating same key, got %d", size)
	}
	got := c.Get(context.Background(), "a")
	if got == nil || got.OrganizationID != mustUUIDs[1].String() {
		t.Fatalf("expected updated entry, got %v", got)
	}
}

func TestCache_InvalidateRemovesEntry(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.Set(context.Background(), testCacheEntry("key-hash-1", mustUUIDs[0].String()))

	c.Invalidate(context.Background(), "key-hash-1")
	if got := c.Get(context.Background(), "key-hash-1"); got != nil {
		t.Fatal("expected entry removed after Invalidate")
	}
}

func TestCache_InvalidateByOrgScansMatchingOrg(t *testing.T) {
	c := NewInMemoryCredentialCache(10, time.Hour)
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("b", mustUUIDs[1].String()))

	c.InvalidateByOrg(context.Background(), mustUUIDs[0].String())

	if got := c.Get(context.Background(), "a"); got != nil {
		t.Fatal("expected org entry to be invalidated")
	}
	if got := c.Get(context.Background(), "b"); got == nil {
		t.Fatal("expected other org entry to survive")
	}
}

func TestCache_StatsReportsSizeAndCapacity(t *testing.T) {
	c := NewInMemoryCredentialCache(5, time.Hour)
	c.Set(context.Background(), testCacheEntry("a", mustUUIDs[0].String()))
	c.Set(context.Background(), testCacheEntry("b", mustUUIDs[0].String()))

	size, maxSize := c.Stats()
	if size != 2 {
		t.Fatalf("expected size 2, got %d", size)
	}
	if maxSize != 5 {
		t.Fatalf("expected max size 5, got %d", maxSize)
	}
}
