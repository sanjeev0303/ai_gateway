package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ConversationRepository manages persisted chat sessions.
type ConversationRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewConversation) (domain.Conversation, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Conversation, error)
	// Detail returns one conversation together with the aggregates derived
	// from its messages; Get returns the bare row.
	Detail(ctx context.Context, orgID, id uuid.UUID) (domain.ConversationSummary, error)
	List(ctx context.Context, orgID uuid.UUID, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, error)
	PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error)
}

// ConversationMessageRepository manages the turns of a conversation.
type ConversationMessageRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewConversationMessage) (domain.ConversationMessage, error)
	List(ctx context.Context, orgID, conversationID uuid.UUID) ([]domain.ConversationMessage, error)
}

// LlmRequestRepository records and reads normalized gateway request rows.
type LlmRequestRepository interface {
	// Record is idempotent per (organization, request_id): a duplicate
	// returns ErrConflict.
	Record(ctx context.Context, orgID uuid.UUID, input domain.NewLlmRequest) (domain.LlmRequest, error)
	GetByRequestID(ctx context.Context, orgID uuid.UUID, requestID string) (domain.LlmRequest, error)
	// List returns requests within the [from, to) window.
	List(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.LlmRequest, error)
	GetRecent(ctx context.Context, orgID uuid.UUID) ([]domain.LlmRequest, error)
	GetRequests(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, error)
	PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error)
}

// UsageEventRepository records and reads the append-only usage ledger.
type UsageEventRepository interface {
	// Record is idempotent per (organization, request_id, event_type): a
	// duplicate returns ErrConflict.
	Record(ctx context.Context, orgID uuid.UUID, input domain.NewUsageEvent) (domain.UsageEvent, error)
	Get(ctx context.Context, orgID uuid.UUID, requestID, eventType string) (domain.UsageEvent, error)
	ListByRequest(ctx context.Context, orgID uuid.UUID, requestID string) ([]domain.UsageEvent, error)
	PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error)
}

// UsageRollupRepository rebuilds the hourly/daily/monthly bucket tables for one
// organization from llm_requests over the half-open window [from, to).
type UsageRollupRepository interface {
	Recompute(ctx context.Context, orgID uuid.UUID, from, to time.Time) error
	GetSummary(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageSummary, error)
	GetTimeSeries(ctx context.Context, orgID uuid.UUID, from, to time.Time, projectID *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error)
	GetByProject(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByProject, error)
	GetByModel(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByModel, error)
	GetByProvider(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByProvider, error)
	GetByUser(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByUser, error)
	GetByServiceAccount(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByServiceAccount, error)
	GetByTeam(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByTeam, error)
	// GetLatency returns percentile timings for the whole filtered set, and
	// GetLatencyByProviderModel breaks the same percentiles down per provider
	// and model. Both compute percentiles in Postgres.
	GetLatency(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageLatency, error)
	GetLatencyByProviderModel(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageLatency, error)
}

// IdempotencyRepository protects request deduplication.
type IdempotencyRepository interface {
	// Create registers a key; a duplicate (scope, key) returns ErrConflict.
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewIdempotencyKey) (domain.IdempotencyKey, error)
	Get(ctx context.Context, orgID uuid.UUID, scopeKey, key string) (domain.IdempotencyKey, error)
}

// PlanRepository manages sellable tiers. Plans are a global table.
type PlanRepository interface {
	Create(ctx context.Context, input domain.NewPlan) (domain.Plan, error)
	List(ctx context.Context) ([]domain.Plan, error)
}

// SubscriptionRepository links organizations to plans.
type SubscriptionRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewSubscription) (domain.Subscription, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Subscription, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Subscription, error)
	// ActiveWithPlan returns the organization's active or trialing subscription
	// joined to its plan, or ErrNotFound when it has none.
	ActiveWithPlan(ctx context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error)
	SetPlan(ctx context.Context, orgID, planID uuid.UUID) error
	Cancel(ctx context.Context, orgID uuid.UUID) error
	UpdatePaymentDetails(ctx context.Context, orgID uuid.UUID, provider, subID, custID, status string) error
}

// BillingEventRepository records the append-only billing ledger.
type BillingEventRepository interface {
	// Record is idempotent per (organization, billing_event_id): a duplicate
	// returns ErrConflict.
	Record(ctx context.Context, orgID uuid.UUID, input domain.NewBillingEvent) (domain.BillingEvent, error)
	Get(ctx context.Context, orgID uuid.UUID, billingEventID string) (domain.BillingEvent, error)
	PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error)
}

// InvoiceRepository reads locally computed billing statements.
type InvoiceRepository interface {
	// List returns the organization's invoices, newest issued first, each with
	// its lines attached.
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Invoice, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Invoice, error)
	RecordPayment(ctx context.Context, orgID uuid.UUID, paymentProvider, paymentID string, amountMicros int64) error
}

// CheckoutRepository manages trusted server-side checkout sessions.
type CheckoutRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewCheckout) (domain.Checkout, error)
	Get(ctx context.Context, orgID uuid.UUID, checkoutSessionID string) (domain.Checkout, error)
	GetByOrderID(ctx context.Context, orgID uuid.UUID, orderID string) (domain.Checkout, error)
	UpdateStatus(ctx context.Context, orgID uuid.UUID, checkoutSessionID, status, paymentID string) error
	ExpirePending(ctx context.Context, olderThan time.Time) (int64, error)
}

// PricingModelRepository manages per-organization model price catalogs.
type PricingModelRepository interface {
	Upsert(ctx context.Context, orgID uuid.UUID, input domain.NewPricingModel) (domain.PricingModel, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.PricingModel, error)
}

// PricingVersionRepository manages versioned price sheets.
type PricingVersionRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewPricingVersion) (domain.PricingVersion, error)
	List(ctx context.Context, orgID, pricingModelID uuid.UUID) ([]domain.PricingVersion, error)
}

// AlertRepository manages configured notification rules.
type AlertRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewAlert) (domain.Alert, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Alert, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Alert, error)
	Update(ctx context.Context, orgID, id uuid.UUID, input domain.AlertUpdate) (domain.Alert, error)
	Delete(ctx context.Context, orgID, id uuid.UUID) error
	UpdateTriggered(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Alert, error)
	CreateEvent(ctx context.Context, orgID uuid.UUID, input domain.NewAlertEvent) (domain.AlertEvent, error)
	ListEvents(ctx context.Context, orgID uuid.UUID, alertID *uuid.UUID, limit, offset int32) ([]domain.AlertEvent, error)
	GetLatestEvent(ctx context.Context, orgID, alertID uuid.UUID) (*domain.AlertEvent, error)
	UpdateEventStatus(ctx context.Context, orgID, eventID uuid.UUID, status string) (domain.AlertEvent, error)
	CreateNotification(ctx context.Context, orgID uuid.UUID, input domain.NewInAppNotification) (domain.InAppNotification, error)
	ListNotifications(ctx context.Context, orgID uuid.UUID, unreadOnly bool, limit, offset int32) ([]domain.InAppNotification, error)
	MarkNotificationRead(ctx context.Context, orgID, id uuid.UUID) error
	MarkAllNotificationsRead(ctx context.Context, orgID uuid.UUID) error
}

// AuditLogRepository appends immutable audit records and reads them back.
type AuditLogRepository interface {
	Append(ctx context.Context, orgID uuid.UUID, input domain.NewAuditLog) (domain.AuditLog, error)
	List(ctx context.Context, orgID uuid.UUID, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.AuditLog, error)
	PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error)
}
