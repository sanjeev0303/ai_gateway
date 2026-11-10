package worker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

// fakeMonitorRepo keeps live alerts in memory, keyed like the unique index.
type fakeMonitorRepo struct {
	ports.PlatformMonitorRepository
	live    map[string]domain.PlatformAlert
	samples []domain.ServiceSample
}

func (f *fakeMonitorRepo) InsertSamples(_ context.Context, s []domain.ServiceSample) error {
	f.samples = append(f.samples, s...)
	return nil
}
func (f *fakeMonitorRepo) DatabaseStats(context.Context) (int, int64, error) { return 3, 1 << 20, nil }
func (f *fakeMonitorRepo) ErrorCounts(context.Context, time.Time) (map[string]int, error) {
	return map[string]int{}, nil
}
func (f *fakeMonitorRepo) Purge(context.Context, time.Time, time.Time, time.Time) error { return nil }
func (f *fakeMonitorRepo) MarkAlertNotified(context.Context, uuid.UUID) error           { return nil }

func (f *fakeMonitorRepo) OpenAlert(_ context.Context, in domain.NewPlatformAlert) (domain.PlatformAlert, bool, error) {
	key := in.Kind + "/" + in.Service
	if a, ok := f.live[key]; ok {
		return a, false, nil
	}
	a := domain.PlatformAlert{ID: uuid.New(), Kind: in.Kind, Service: in.Service, Severity: in.Severity, Title: in.Title, Status: "open", OpenedAt: time.Now()}
	f.live[key] = a
	return a, true, nil
}

func (f *fakeMonitorRepo) ResolveAlert(_ context.Context, kind, service string) (domain.PlatformAlert, bool, error) {
	key := kind + "/" + service
	a, ok := f.live[key]
	if !ok {
		return a, false, nil
	}
	delete(f.live, key)
	return a, true, nil
}

type fakeMailer struct{ subjects []string }

func (f *fakeMailer) Send(_ []string, subject, _ string) error {
	f.subjects = append(f.subjects, subject)
	return nil
}

func TestMonitor_DownAlertOpensOnceAndResolves(t *testing.T) {
	var healthy atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := &fakeMonitorRepo{live: map[string]domain.PlatformAlert{}}
	mailer := &fakeMailer{}
	m := NewMonitor(MonitorDeps{
		Repo:        repo,
		Targets:     []MonitorTarget{{Name: "gateway-v2", URL: server.URL}},
		Mailer:      mailer,
		AdminEmails: []string{"root@example.com"},
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ctx := context.Background()

	m.Poll(ctx) // first failure: not yet an outage
	if len(repo.live) != 0 {
		t.Fatalf("alert opened after one failed check: %v", repo.live)
	}
	m.Poll(ctx) // second failure: down
	m.Poll(ctx) // still down: same alert, no second email
	if _, ok := repo.live["service_down/gateway-v2"]; !ok {
		t.Fatal("no service_down alert after two failed checks")
	}
	if len(mailer.subjects) != 1 || !strings.Contains(mailer.subjects[0], "CRITICAL") {
		t.Fatalf("expected one critical email, got %v", mailer.subjects)
	}

	healthy.Store(true)
	m.Poll(ctx)
	if len(repo.live) != 0 {
		t.Fatalf("alert not resolved after recovery: %v", repo.live)
	}
	if len(mailer.subjects) != 2 || !strings.Contains(mailer.subjects[1], "Resolved") {
		t.Fatalf("expected a resolved email, got %v", mailer.subjects)
	}
}

func TestParseMonitorTargets(t *testing.T) {
	targets, err := ParseMonitorTargets([]string{"gateway-v2=http://gateway-v2:8080/", " control-api = http://control-api:8080 "})
	if err != nil || len(targets) != 2 || targets[0].URL != "http://gateway-v2:8080" || targets[1].Name != "control-api" {
		t.Fatalf("unexpected parse: %+v %v", targets, err)
	}
	if _, err := ParseMonitorTargets([]string{"no-url"}); err == nil {
		t.Fatal("accepted a target without a URL")
	}
}
