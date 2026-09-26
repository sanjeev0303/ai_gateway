package domain

import (
	"time"

	"github.com/google/uuid"
)

// Model is one entry in the DB-driven model catalog. The catalog is global
// (no organization_id): adding a model is a row, not a code change.
type Model struct {
	ID          uuid.UUID
	ProviderKind string
	ModelSlug   string
	DisplayName string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Model statuses, mirroring the CHECK on the catalog status column.
const (
	ModelStatusActive   = "active"
	ModelStatusInactive = "inactive"
)

// ModelMetadata is the governance-owned metadata of one model catalog entry.
// Providers advertise only facts (providerclient.Model); the platform layers
// governance fields — status, pricing version and feature gates — on top when
// seeding the catalog. Feature gates a provider does not advertise are seeded
// by platform policy, never invented by the provider client.
type ModelMetadata struct {
	ProviderKind      string
	ModelSlug         string
	DisplayName       string
	Capabilities      []string
	ContextLimit      int
	SupportsStreaming bool
	SupportsTools     bool
	SupportsReasoning bool
	Status            string
	PricingVersion    string
}

// Provider health states, mirroring the CHECK on the provider health column.
const (
	ProviderHealthUnknown   = "unknown"
	ProviderHealthHealthy   = "healthy"
	ProviderHealthDegraded  = "degraded"
	ProviderHealthUnhealthy = "unhealthy"
)

// ProviderHealthState captures the last observed connectivity state of an
// upstream provider account. StatusReason carries a stable, machine-readable
// reason code (e.g. "upstream_4xx", "dns", "timeout"); LastError carries the
// human-readable detail and must never contain secrets or request payloads.
type ProviderHealthState struct {
	ProviderKind string
	Status       string
	StatusReason string
	LastError    string
	CheckedAt    time.Time
}

// Healthy reports whether the state admits routing traffic (healthy, or
// degraded which is still servable).
func (s ProviderHealthState) Healthy() bool {
	return s.Status == ProviderHealthHealthy || s.Status == ProviderHealthDegraded
}