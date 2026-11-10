package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type platformMonitorRepo struct {
	store *postgres.Store
}

// NewPlatformMonitorRepository stores the platform monitor's samples, service
// log lines and alerts. They are platform tables, outside tenant RLS.
func NewPlatformMonitorRepository(store *postgres.Store) ports.PlatformMonitorRepository {
	return &platformMonitorRepo{store: store}
}

const sampleColumns = `service, sampled_at, up, error, cpu_percent, rss_bytes, heap_bytes, goroutines, detail`

func scanSample(row pgx.Row) (domain.ServiceSample, error) {
	var s domain.ServiceSample
	err := row.Scan(&s.Service, &s.SampledAt, &s.Up, &s.Error, &s.CPUPercent, &s.RSSBytes, &s.HeapBytes, &s.Goroutines, &s.Detail)
	return s, err
}

func (r *platformMonitorRepo) InsertSamples(ctx context.Context, samples []domain.ServiceSample) error {
	batch := &pgx.Batch{}
	for _, s := range samples {
		detail := s.Detail
		if len(detail) == 0 {
			detail = []byte("{}")
		}
		batch.Queue(`INSERT INTO service_samples (`+sampleColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			s.Service, s.SampledAt, s.Up, s.Error, s.CPUPercent, s.RSSBytes, s.HeapBytes, s.Goroutines, detail)
	}
	if err := r.store.Pool().SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("insert service samples: %w", err)
	}
	return nil
}

func (r *platformMonitorRepo) LatestSamples(ctx context.Context) ([]domain.ServiceSample, error) {
	return r.samples(ctx, `SELECT DISTINCT ON (service) `+sampleColumns+` FROM service_samples ORDER BY service, sampled_at DESC`)
}

func (r *platformMonitorRepo) SampleSeries(ctx context.Context, since time.Time) ([]domain.ServiceSample, error) {
	return r.samples(ctx, `SELECT `+sampleColumns+` FROM service_samples WHERE sampled_at >= $1 ORDER BY sampled_at`, since)
}

func (r *platformMonitorRepo) samples(ctx context.Context, sql string, args ...any) ([]domain.ServiceSample, error) {
	rows, err := r.store.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("read service samples: %w", err)
	}
	defer rows.Close()
	out := []domain.ServiceSample{}
	for rows.Next() {
		s, err := scanSample(rows)
		if err != nil {
			return nil, fmt.Errorf("scan service sample: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *platformMonitorRepo) ErrorCounts(ctx context.Context, since time.Time) (map[string]int, error) {
	rows, err := r.store.Pool().Query(ctx,
		`SELECT service, count(*) FROM service_logs WHERE level = 'ERROR' AND logged_at >= $1 GROUP BY service`, since)
	if err != nil {
		return nil, fmt.Errorf("count service errors: %w", err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var service string
		var n int
		if err := rows.Scan(&service, &n); err != nil {
			return nil, err
		}
		counts[service] = n
	}
	return counts, rows.Err()
}

func (r *platformMonitorRepo) ListLogs(ctx context.Context, f domain.ServiceLogFilter, limit, offset int32) ([]domain.ServiceLog, int64, error) {
	rows, err := r.store.Pool().Query(ctx, `
		SELECT id, service, level, message, attrs, logged_at, count(*) OVER ()
		FROM service_logs
		WHERE ($1 = '' OR service = $1)
		  AND ($2 = '' OR level = $2)
		  AND ($3 = '' OR message ILIKE '%' || $3 || '%')
		ORDER BY logged_at DESC, id DESC
		LIMIT $4 OFFSET $5`, f.Service, f.Level, f.Search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list service logs: %w", err)
	}
	defer rows.Close()
	logs := []domain.ServiceLog{}
	var total int64
	for rows.Next() {
		var l domain.ServiceLog
		if err := rows.Scan(&l.ID, &l.Service, &l.Level, &l.Message, &l.Attrs, &l.LoggedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan service log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

func (r *platformMonitorRepo) DatabaseStats(ctx context.Context) (int, int64, error) {
	var connections int
	var size int64
	err := r.store.Pool().QueryRow(ctx, `
		SELECT (SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()),
		       pg_database_size(current_database())`).Scan(&connections, &size)
	return connections, size, err
}

const alertColumns = `id, kind, service, severity, title, detail, status, opened_at, last_seen_at,
	acknowledged_at, acknowledged_by, resolved_at, notified_at`

func scanAlert(row pgx.Row) (domain.PlatformAlert, error) {
	var a domain.PlatformAlert
	err := row.Scan(&a.ID, &a.Kind, &a.Service, &a.Severity, &a.Title, &a.Detail, &a.Status, &a.OpenedAt,
		&a.LastSeenAt, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.NotifiedAt)
	return a, err
}

func (r *platformMonitorRepo) OpenAlert(ctx context.Context, in domain.NewPlatformAlert) (domain.PlatformAlert, bool, error) {
	// xmax is 0 only on a freshly inserted row, which is how a new alert is
	// told apart from a refreshed one in a single statement.
	row := r.store.Pool().QueryRow(ctx, `
		INSERT INTO platform_alerts (kind, service, severity, title, detail)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kind, service) WHERE status <> 'resolved'
		DO UPDATE SET last_seen_at = now(), detail = EXCLUDED.detail, title = EXCLUDED.title, severity = EXCLUDED.severity
		RETURNING `+alertColumns+`, (xmax = 0)`, in.Kind, in.Service, in.Severity, in.Title, in.Detail)
	var a domain.PlatformAlert
	var created bool
	err := row.Scan(&a.ID, &a.Kind, &a.Service, &a.Severity, &a.Title, &a.Detail, &a.Status, &a.OpenedAt,
		&a.LastSeenAt, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.NotifiedAt, &created)
	if err != nil {
		return a, false, fmt.Errorf("open platform alert: %w", err)
	}
	return a, created, nil
}

func (r *platformMonitorRepo) ResolveAlert(ctx context.Context, kind, service string) (domain.PlatformAlert, bool, error) {
	a, err := scanAlert(r.store.Pool().QueryRow(ctx, `
		UPDATE platform_alerts SET status = 'resolved', resolved_at = now()
		WHERE kind = $1 AND service = $2 AND status <> 'resolved'
		RETURNING `+alertColumns, kind, service))
	if errors.Is(err, pgx.ErrNoRows) {
		return a, false, nil
	}
	if err != nil {
		return a, false, fmt.Errorf("resolve platform alert: %w", err)
	}
	return a, true, nil
}

func (r *platformMonitorRepo) MarkAlertNotified(ctx context.Context, id uuid.UUID) error {
	_, err := r.store.Pool().Exec(ctx, `UPDATE platform_alerts SET notified_at = now() WHERE id = $1`, id)
	return err
}

func (r *platformMonitorRepo) ListAlerts(ctx context.Context, status string, limit, offset int32) ([]domain.PlatformAlert, int64, error) {
	// "live" means open or acknowledged: everything not yet resolved.
	rows, err := r.store.Pool().Query(ctx, `
		SELECT `+alertColumns+`, count(*) OVER ()
		FROM platform_alerts
		WHERE $1 = '' OR status = $1 OR ($1 = 'live' AND status <> 'resolved')
		ORDER BY (status = 'resolved'), opened_at DESC
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list platform alerts: %w", err)
	}
	defer rows.Close()
	alerts := []domain.PlatformAlert{}
	var total int64
	for rows.Next() {
		var a domain.PlatformAlert
		if err := rows.Scan(&a.ID, &a.Kind, &a.Service, &a.Severity, &a.Title, &a.Detail, &a.Status, &a.OpenedAt,
			&a.LastSeenAt, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.NotifiedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan platform alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	return alerts, total, rows.Err()
}

func (r *platformMonitorRepo) AcknowledgeAlert(ctx context.Context, id uuid.UUID, by string) error {
	tag, err := r.store.Pool().Exec(ctx, `
		UPDATE platform_alerts SET status = 'acknowledged', acknowledged_at = now(), acknowledged_by = $2
		WHERE id = $1 AND status = 'open'`, id, by)
	if err != nil {
		return fmt.Errorf("acknowledge platform alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *platformMonitorRepo) CountLiveAlerts(ctx context.Context) (int64, error) {
	var n int64
	err := r.store.Pool().QueryRow(ctx, `SELECT count(*) FROM platform_alerts WHERE status <> 'resolved'`).Scan(&n)
	return n, err
}

func (r *platformMonitorRepo) Purge(ctx context.Context, samplesBefore, logsBefore, alertsBefore time.Time) error {
	batch := &pgx.Batch{}
	batch.Queue(`DELETE FROM service_samples WHERE sampled_at < $1`, samplesBefore)
	batch.Queue(`DELETE FROM service_logs WHERE logged_at < $1`, logsBefore)
	batch.Queue(`DELETE FROM platform_alerts WHERE status = 'resolved' AND resolved_at < $1`, alertsBefore)
	if err := r.store.Pool().SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("purge platform monitoring: %w", err)
	}
	return nil
}
