package invalidation

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// PolicyInvalidationChannel is the Redis pub/sub channel for real-time policy and budget invalidations.
	PolicyInvalidationChannel = "aigw:invalidation:policy"

	// CredentialInvalidationChannel is the Redis pub/sub channel for real-time credential revocation.
	CredentialInvalidationChannel = "aigw:invalidation:credential"
)

// PolicyInvalidationPayload carries the target organization ID for policy snapshot eviction.
type PolicyInvalidationPayload struct {
	OrganizationID string `json:"org_id"`
}

// CredentialInvalidationPayload carries the organization and credential ID for immediate cache eviction.
type CredentialInvalidationPayload struct {
	OrganizationID string `json:"org_id"`
	CredentialID   string `json:"credential_id"`
}

// RedisInvalidator broadcasts invalidation events across processes via Redis Pub/Sub,
// and notifies local in-process listeners immediately (<1ms).
type RedisInvalidator struct {
	client    *redis.Client
	logger    *slog.Logger
	mu        sync.RWMutex
	listeners []func(orgID uuid.UUID)
}

// NewRedisInvalidator creates an invalidator. If client is nil, it acts as an in-process invalidator.
func NewRedisInvalidator(client *redis.Client, logger *slog.Logger) *RedisInvalidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisInvalidator{
		client: client,
		logger: logger,
	}
}

// AddListener registers a local callback invoked on Invalidate.
func (r *RedisInvalidator) AddListener(fn func(orgID uuid.UUID)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = append(r.listeners, fn)
}

// Invalidate publishes an invalidation event for the given organization ID.
// Implements policy.PolicyInvalidator and budget.BudgetInvalidator.
func (r *RedisInvalidator) Invalidate(orgID uuid.UUID) {
	// 1. Notify in-process listeners
	r.mu.RLock()
	for _, fn := range r.listeners {
		fn(orgID)
	}
	r.mu.RUnlock()

	// 2. Publish to Redis Pub/Sub for distributed gateway replicas
	if r.client != nil {
		payload, err := json.Marshal(PolicyInvalidationPayload{OrganizationID: orgID.String()})
		if err == nil {
			ctx := context.Background()
			if err := r.client.Publish(ctx, PolicyInvalidationChannel, payload).Err(); err != nil {
				r.logger.Warn("failed to publish policy invalidation to redis", "org_id", orgID, "error", err)
			} else {
				r.logger.Debug("published policy invalidation event", "org_id", orgID)
			}
		}
	}
}

// InvalidateCredential broadcasts immediate credential revocation across processes.
func (r *RedisInvalidator) InvalidateCredential(orgID, credID uuid.UUID) {
	if r.client != nil {
		payload, err := json.Marshal(CredentialInvalidationPayload{
			OrganizationID: orgID.String(),
			CredentialID:   credID.String(),
		})
		if err == nil {
			ctx := context.Background()
			_ = r.client.Publish(ctx, CredentialInvalidationChannel, payload).Err()
		}
	}
}
