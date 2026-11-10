package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/runtimestats"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

// Alert thresholds. A service must fail two polls in a row before it counts
// as down, and run hot for three, so one slow answer never pages anyone.
const (
	downAfterPolls    = 2
	hotCPUPercent     = 90.0
	hotAfterPolls     = 3
	errorSpikeCount   = 20
	errorSpikeWindow  = 5 * time.Minute
	outboxStaleAfter  = 5 * time.Minute
	purgeEvery        = time.Hour
	keepSamples       = 7 * 24 * time.Hour
	keepLogs          = 14 * 24 * time.Hour
	keepResolvedAlert = 90 * 24 * time.Hour
)

// MonitorTarget is a service the monitor polls over HTTP.
type MonitorTarget struct {
	Name string
	URL  string
}

// Mailer sends an alert email.
type Mailer interface {
	Send(to []string, subject, body string) error
}

type MonitorDeps struct {
	Repo    ports.PlatformMonitorRepository
	Outbox  func(ctx context.Context) (domain.PlatformOutbox, error)
	Pool    *pgxpool.Pool
	Targets []MonitorTarget
	Token   string
	// Redis probes Redis when it is enabled; nil skips it.
	Redis func(ctx context.Context) (map[string]any, error)
	// Mailer is nil when SMTP is not configured: alerts stay in the console.
	Mailer      Mailer
	AdminEmails []string
	// ConsoleURL is linked from alert emails.
	ConsoleURL       string
	MemoryAlertBytes int64
	Interval         time.Duration
	Log              *slog.Logger
}

type cpuReading struct {
	seconds float64
	at      time.Time
}

// Monitor keeps the per-service state that spans polls.
type Monitor struct {
	deps      MonitorDeps
	client    *http.Client
	lastCPU   map[string]cpuReading
	downPolls map[string]int
	hotPolls  map[string]int
	lastPurge time.Time
}

func NewMonitor(deps MonitorDeps) *Monitor {
	if deps.Interval <= 0 {
		deps.Interval = 30 * time.Second
	}
	return &Monitor{
		deps:      deps,
		client:    &http.Client{Timeout: 5 * time.Second},
		lastCPU:   map[string]cpuReading{},
		downPolls: map[string]int{},
		hotPolls:  map[string]int{},
	}
}

// Run polls until ctx ends.
func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.deps.Interval)
	defer ticker.Stop()
	m.Poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Poll(ctx)
		}
	}
}

// Poll takes one round of samples and evaluates the alert rules on them.
func (m *Monitor) Poll(ctx context.Context) {
	now := time.Now()
	samples := make([]domain.ServiceSample, 0, len(m.deps.Targets)+3)
	for _, target := range m.deps.Targets {
		samples = append(samples, m.probeService(ctx, target, now))
	}
	samples = append(samples, m.selfSample(now), m.postgresSample(ctx, now))
	if m.deps.Redis != nil {
		samples = append(samples, m.redisSample(ctx, now))
	}

	if err := m.deps.Repo.InsertSamples(ctx, samples); err != nil {
		m.deps.Log.Error("monitor: store samples", "error", err)
	}
	for _, s := range samples {
		m.checkSample(ctx, s)
	}
	m.checkErrorSpikes(ctx, samples, now)
	m.checkOutbox(ctx)

	if now.Sub(m.lastPurge) >= purgeEvery {
		m.lastPurge = now
		if err := m.deps.Repo.Purge(ctx, now.Add(-keepSamples), now.Add(-keepLogs), now.Add(-keepResolvedAlert)); err != nil {
			m.deps.Log.Warn("monitor: purge old monitoring data", "error", err)
		}
	}
}

func (m *Monitor) probeService(ctx context.Context, target MonitorTarget, now time.Time) domain.ServiceSample {
	sample := domain.ServiceSample{Service: target.Name, SampledAt: now}

	// /readyz checks the service's own dependencies, so "up" means able to serve.
	status, body, err := m.get(ctx, target.URL+"/readyz", "")
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	if status != http.StatusOK {
		sample.Error = fmt.Sprintf("readiness check answered %d: %s", status, strings.TrimSpace(body))
		return sample
	}
	sample.Up = true

	if m.deps.Token == "" {
		return sample
	}
	status, body, err = m.get(ctx, target.URL+"/internal/stats", m.deps.Token)
	if err != nil || status != http.StatusOK {
		// Up but not reporting resources: keep the sample, say why.
		sample.Error = "stats unavailable"
		return sample
	}
	var stats runtimestats.Stats
	if err := json.Unmarshal([]byte(body), &stats); err != nil {
		sample.Error = "stats unreadable"
		return sample
	}
	m.fillFromStats(&sample, stats)
	return sample
}

func (m *Monitor) selfSample(now time.Time) domain.ServiceSample {
	sample := domain.ServiceSample{Service: "worker", SampledAt: now, Up: true}
	m.fillFromStats(&sample, runtimestats.Collect(m.deps.Pool))
	return sample
}

func (m *Monitor) fillFromStats(sample *domain.ServiceSample, stats runtimestats.Stats) {
	rss, heap := stats.RSSBytes, stats.HeapBytes
	sample.RSSBytes, sample.HeapBytes = &rss, &heap
	// Only Go services have goroutines; a Node app (the dashboard) reports none.
	if stats.Goroutines > 0 {
		goroutines := stats.Goroutines
		sample.Goroutines = &goroutines
	}

	// CPU percent is of the whole machine's capacity, from the change in CPU
	// seconds since the previous poll. The first poll has nothing to compare.
	if prev, ok := m.lastCPU[sample.Service]; ok && stats.NumCPU > 0 {
		wall := sample.SampledAt.Sub(prev.at).Seconds()
		if wall > 0 && stats.CPUSeconds >= prev.seconds {
			pct := (stats.CPUSeconds - prev.seconds) / (wall * float64(stats.NumCPU)) * 100
			sample.CPUPercent = &pct
		}
	}
	m.lastCPU[sample.Service] = cpuReading{seconds: stats.CPUSeconds, at: sample.SampledAt}

	detail, _ := json.Marshal(map[string]any{
		"uptime_seconds": int64(stats.UptimeSeconds),
		"num_cpu":        stats.NumCPU,
		"gc_cycles":      stats.GCCycles,
		"db_pool":        stats.DBPool,
	})
	sample.Detail = detail
}

func (m *Monitor) postgresSample(ctx context.Context, now time.Time) domain.ServiceSample {
	sample := domain.ServiceSample{Service: "postgres", SampledAt: now}
	connections, size, err := m.deps.Repo.DatabaseStats(ctx)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	sample.Up = true
	sample.Detail, _ = json.Marshal(map[string]any{"connections": connections, "size_bytes": size})
	return sample
}

func (m *Monitor) redisSample(ctx context.Context, now time.Time) domain.ServiceSample {
	sample := domain.ServiceSample{Service: "redis", SampledAt: now}
	detail, err := m.deps.Redis(ctx)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	sample.Up = true
	if used, ok := detail["used_memory"].(int64); ok {
		sample.RSSBytes = &used
	}
	sample.Detail, _ = json.Marshal(detail)
	return sample
}

func (m *Monitor) get(ctx context.Context, url, token string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, string(body), nil
}

// checkSample applies the per-service rules: down, CPU, memory.
func (m *Monitor) checkSample(ctx context.Context, s domain.ServiceSample) {
	if !s.Up {
		m.downPolls[s.Service]++
		if m.downPolls[s.Service] >= downAfterPolls {
			m.open(ctx, domain.NewPlatformAlert{
				Kind: "service_down", Service: s.Service, Severity: domain.PlatformAlertCritical,
				Title:  s.Service + " is down",
				Detail: fmt.Sprintf("Failed %d checks in a row. Last error: %s", m.downPolls[s.Service], s.Error),
			})
		}
		return
	}
	m.downPolls[s.Service] = 0
	m.resolve(ctx, "service_down", s.Service)

	if s.CPUPercent != nil && *s.CPUPercent >= hotCPUPercent {
		m.hotPolls[s.Service]++
		if m.hotPolls[s.Service] >= hotAfterPolls {
			m.open(ctx, domain.NewPlatformAlert{
				Kind: "high_cpu", Service: s.Service, Severity: domain.PlatformAlertWarning,
				Title:  fmt.Sprintf("%s CPU above %.0f%%", s.Service, hotCPUPercent),
				Detail: fmt.Sprintf("%.0f%% of machine capacity for %d checks in a row.", *s.CPUPercent, m.hotPolls[s.Service]),
			})
		}
	} else {
		m.hotPolls[s.Service] = 0
		m.resolve(ctx, "high_cpu", s.Service)
	}

	// Redis reports its dataset size, not a process, so it is left out here.
	if s.Service != "redis" && s.RSSBytes != nil && m.deps.MemoryAlertBytes > 0 {
		if *s.RSSBytes >= m.deps.MemoryAlertBytes {
			m.open(ctx, domain.NewPlatformAlert{
				Kind: "high_memory", Service: s.Service, Severity: domain.PlatformAlertWarning,
				Title:  s.Service + " memory is high",
				Detail: fmt.Sprintf("Resident memory %d MB, alert threshold %d MB.", *s.RSSBytes>>20, m.deps.MemoryAlertBytes>>20),
			})
		} else {
			m.resolve(ctx, "high_memory", s.Service)
		}
	}
}

func (m *Monitor) checkErrorSpikes(ctx context.Context, samples []domain.ServiceSample, now time.Time) {
	counts, err := m.deps.Repo.ErrorCounts(ctx, now.Add(-errorSpikeWindow))
	if err != nil {
		m.deps.Log.Warn("monitor: count error logs", "error", err)
		return
	}
	services := map[string]bool{}
	for _, s := range samples {
		services[s.Service] = true
	}
	for service := range counts {
		services[service] = true
	}
	for service := range services {
		if n := counts[service]; n >= errorSpikeCount {
			m.open(ctx, domain.NewPlatformAlert{
				Kind: "error_spike", Service: service, Severity: domain.PlatformAlertWarning,
				Title:  fmt.Sprintf("%s logged %d errors in 5 minutes", service, n),
				Detail: "Open Logs in the admin console, filtered to this service, to see them.",
			})
		} else {
			m.resolve(ctx, "error_spike", service)
		}
	}
}

func (m *Monitor) checkOutbox(ctx context.Context) {
	if m.deps.Outbox == nil {
		return
	}
	outbox, err := m.deps.Outbox(ctx)
	if err != nil {
		m.deps.Log.Warn("monitor: read outbox", "error", err)
		return
	}
	if time.Duration(outbox.OldestAgeSeconds)*time.Second >= outboxStaleAfter {
		m.open(ctx, domain.NewPlatformAlert{
			Kind: "outbox_backlog", Service: "worker", Severity: domain.PlatformAlertWarning,
			Title:  "Usage and security events are backing up",
			Detail: fmt.Sprintf("%d events waiting, the oldest for %d minutes. Last error: %s", outbox.Pending, outbox.OldestAgeSeconds/60, outbox.LastError),
		})
		return
	}
	m.resolve(ctx, "outbox_backlog", "worker")
}

func (m *Monitor) open(ctx context.Context, in domain.NewPlatformAlert) {
	alert, created, err := m.deps.Repo.OpenAlert(ctx, in)
	if err != nil {
		m.deps.Log.Error("monitor: open alert", "kind", in.Kind, "service", in.Service, "error", err)
		return
	}
	if created {
		m.notify(ctx, alert, "Opened")
	}
}

func (m *Monitor) resolve(ctx context.Context, kind, service string) {
	alert, resolved, err := m.deps.Repo.ResolveAlert(ctx, kind, service)
	if err != nil {
		m.deps.Log.Error("monitor: resolve alert", "kind", kind, "service", service, "error", err)
		return
	}
	if resolved {
		m.notify(ctx, alert, "Resolved")
	}
}

// notify emails the admins. A failure is logged but never blocks the monitor;
// the alert is in the console either way.
func (m *Monitor) notify(ctx context.Context, alert domain.PlatformAlert, change string) {
	if m.deps.Mailer == nil || len(m.deps.AdminEmails) == 0 {
		return
	}
	subject := fmt.Sprintf("[Tollseal %s] %s: %s", strings.ToUpper(alert.Severity), change, alert.Title)
	body := fmt.Sprintf("%s: %s\n\nService: %s\nSeverity: %s\nOpened: %s\n\n%s\n",
		change, alert.Title, alert.Service, alert.Severity, alert.OpenedAt.UTC().Format(time.RFC1123), alert.Detail)
	if m.deps.ConsoleURL != "" {
		body += "\nAlerts: " + strings.TrimRight(m.deps.ConsoleURL, "/") + "/alerts\n"
	}
	if err := m.deps.Mailer.Send(m.deps.AdminEmails, subject, body); err != nil {
		m.deps.Log.Warn("monitor: email alert", "alert_id", alert.ID, "error", err)
		return
	}
	if change == "Opened" {
		if err := m.deps.Repo.MarkAlertNotified(ctx, alert.ID); err != nil {
			m.deps.Log.Warn("monitor: mark alert notified", "alert_id", alert.ID, "error", err)
		}
	}
}

// ParseMonitorTargets reads "name=url,name=url" (MONITOR_TARGETS).
func ParseMonitorTargets(raw []string) ([]MonitorTarget, error) {
	targets := make([]MonitorTarget, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		name, url, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(name) == "" || !strings.HasPrefix(strings.TrimSpace(url), "http") {
			return nil, fmt.Errorf("monitor target %q: want name=http://host:port", item)
		}
		targets = append(targets, MonitorTarget{Name: strings.TrimSpace(name), URL: strings.TrimRight(strings.TrimSpace(url), "/")})
	}
	return targets, nil
}
