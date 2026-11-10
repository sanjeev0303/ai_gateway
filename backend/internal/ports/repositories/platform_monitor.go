package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// PlatformMonitorRepository stores what the platform monitor sees: resource
// samples, service log lines and the alerts raised on them (migration 00044).
type PlatformMonitorRepository interface {
	InsertSamples(ctx context.Context, samples []domain.ServiceSample) error
	// LatestSamples returns the newest sample of every service.
	LatestSamples(ctx context.Context) ([]domain.ServiceSample, error)
	// SampleSeries returns every sample since, oldest first.
	SampleSeries(ctx context.Context, since time.Time) ([]domain.ServiceSample, error)
	// ErrorCounts counts ERROR log lines per service since a time.
	ErrorCounts(ctx context.Context, since time.Time) (map[string]int, error)
	ListLogs(ctx context.Context, filter domain.ServiceLogFilter, limit, offset int32) ([]domain.ServiceLog, int64, error)
	// DatabaseStats reports the platform database's own connections and size.
	DatabaseStats(ctx context.Context) (connections int, sizeBytes int64, err error)

	// OpenAlert raises an alert, or refreshes the one already live for the same
	// kind and service. created is true only for a new alert.
	OpenAlert(ctx context.Context, alert domain.NewPlatformAlert) (domain.PlatformAlert, bool, error)
	// ResolveAlert closes the live alert for a kind and service, if any.
	ResolveAlert(ctx context.Context, kind, service string) (domain.PlatformAlert, bool, error)
	MarkAlertNotified(ctx context.Context, id uuid.UUID) error
	ListAlerts(ctx context.Context, status string, limit, offset int32) ([]domain.PlatformAlert, int64, error)
	// AcknowledgeAlert marks an open alert as seen. ErrNotFound when there is no
	// open alert with that ID.
	AcknowledgeAlert(ctx context.Context, id uuid.UUID, by string) error
	CountLiveAlerts(ctx context.Context) (int64, error)

	// Purge deletes samples, logs and resolved alerts older than their cutoffs.
	Purge(ctx context.Context, samplesBefore, logsBefore, alertsBefore time.Time) error
}
