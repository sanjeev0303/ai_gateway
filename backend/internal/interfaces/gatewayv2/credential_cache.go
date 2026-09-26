// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"sync"
	"time"
)

// CredentialCacheEntry represents a cached credential for fast lookup.
type CredentialCacheEntry struct {
	ID             string
	OrganizationID string
	SubjectType    string
	SubjectID      string
	TeamID         *string
	ProjectID      *string
	Scopes         []string
	Status         string
	ExpiresAt      *time.Time
	UpdatedAt      time.Time
}

// InMemoryCredentialCache is a high-performance, thread-safe LRU cache
// for gateway credentials. Optimized for the hot path with minimal allocations.
type InMemoryCredentialCache struct {
	mu       sync.RWMutex
	entries  map[string]*CredentialCacheEntry // key: hash
	lru      *lruList
	maxSize  int
	ttl      time.Duration
	clock    func() time.Time
}

// lruList is a simple doubly-linked list for LRU eviction.
type lruList struct {
	head, tail *lruNode
	len        int
}

type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// NewInMemoryCredentialCache creates a new in-memory credential cache.
func NewInMemoryCredentialCache(maxSize int, ttl time.Duration) *InMemoryCredentialCache {
	c := &InMemoryCredentialCache{
		entries: make(map[string]*CredentialCacheEntry, maxSize),
		lru:     &lruList{},
		maxSize: maxSize,
		ttl:     ttl,
		clock:   time.Now,
	}
	return c
}

// Get retrieves a credential by hash.
func (c *InMemoryCredentialCache) Get(ctx context.Context, keyHash string) *CredentialCacheEntry {
	c.mu.RLock()
	entry, ok := c.entries[keyHash]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	// Check TTL
	if c.clock().Sub(entry.UpdatedAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, keyHash)
		c.lru.remove(keyHash)
		c.mu.Unlock()
		return nil
	}

	// Check expiry
	if entry.ExpiresAt != nil && entry.ExpiresAt.Before(c.clock()) {
		c.mu.Lock()
		delete(c.entries, keyHash)
		c.lru.remove(keyHash)
		c.mu.Unlock()
		return nil
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.lru.moveToFront(keyHash)
	c.mu.Unlock()

	return entry
}

// Set adds or updates a credential in the cache.
func (c *InMemoryCredentialCache) Set(ctx context.Context, entry *CredentialCacheEntry) {
	if entry == nil {
		return
	}

	keyHash := entry.ID // Using ID as key for simplicity, in practice use key hash

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if len(c.entries) >= c.maxSize && c.entries[keyHash] == nil {
		c.evictLRU()
	}

	entry.UpdatedAt = c.clock()
	c.entries[keyHash] = entry
	c.lru.addToFront(keyHash)
}

// Invalidate removes a credential from the cache.
func (c *InMemoryCredentialCache) Invalidate(ctx context.Context, keyHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, keyHash)
	c.lru.remove(keyHash)
}

// InvalidateByOrg clears all credentials for an organization.
func (c *InMemoryCredentialCache) InvalidateByOrg(ctx context.Context, orgID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Since we index by hash, we need to scan - this is a trade-off
	// For production, consider a secondary index by orgID
	for key, entry := range c.entries {
		if entry.OrganizationID == orgID {
			delete(c.entries, key)
			c.lru.remove(key)
		}
	}
}

// Stats returns cache statistics.
func (c *InMemoryCredentialCache) Stats() (size int, maxSize int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}

// evictLRU removes the least recently used entry.
func (c *InMemoryCredentialCache) evictLRU() {
	if c.lru.tail == nil {
		return
	}
	key := c.lru.tail.key
	delete(c.entries, key)
	c.lru.remove(key)
}

// addToFront adds a node to the front of the LRU list.
func (l *lruList) addToFront(key string) {
	node := &lruNode{key: key}
	if l.head == nil {
		l.head = node
		l.tail = node
	} else {
		node.next = l.head
		l.head.prev = node
		l.head = node
	}
	l.len++
}

// moveToFront moves a node to the front of the LRU list.
func (l *lruList) moveToFront(key string) {
	if l.head != nil && l.head.key == key {
		return // already at front
	}

	// Find and remove the node
	node := l.head
	for node != nil {
		if node.key == key {
			l.removeNode(node)
			l.addToFront(key)
			return
		}
		node = node.next
	}
}

// remove removes a node from the LRU list by key.
func (l *lruList) remove(key string) {
	node := l.head
	for node != nil {
		if node.key == key {
			l.removeNode(node)
			return
		}
		node = node.next
	}
}

// removeNode removes a specific node from the LRU list.
func (l *lruList) removeNode(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}

	l.len--
}