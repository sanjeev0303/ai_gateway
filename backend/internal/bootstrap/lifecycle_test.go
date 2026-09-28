package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/infrastructure/eventbus"
	"ai-governance-saas/backend/internal/interfaces/worker"
	"ai-governance-saas/backend/internal/platform/config"
)

func waitForJobs(t *testing.T, jobs *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		jobs.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("jobs.Wait() hung after a failed listen")
	}
}

func TestGatewayV2ConsumerStopsOnListenFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var jobs sync.WaitGroup

	bus := eventbus.NewInMemory(4, slog.New(slog.DiscardHandler))
	jobs.Add(1)
	go func() {
		defer jobs.Done()
		worker.RunUsageConsumer(ctx, bus, nil, slog.New(slog.DiscardHandler))
	}()

	// Stands in for serveHTTP returning a bind error with ctx still live.
	cancel()
	waitForJobs(t, &jobs)
}

func TestWorkerRollupStopsOnListenFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var jobs sync.WaitGroup

	jobs.Add(1)
	go func() {
		defer jobs.Done()
		<-ctx.Done()
	}()

	cancel()
	waitForJobs(t, &jobs)
}

func TestNewUsageBus_RejectsUnimplementedKinds(t *testing.T) {
	quiet := slog.New(slog.DiscardHandler)
	for _, kind := range []string{"postgres", "kafka", "redis", ""} {
		cfg := config.EventbusConfig{Enabled: true, Kind: kind}
		bus, err := newUsageBus(context.Background(), &cfg, quiet)
		if err == nil {
			t.Fatalf("kind %q must be rejected explicitly, got bus %v", kind, bus)
		}
		if bus.EventBus != nil {
			t.Fatalf("kind %q must not yield a usable bus on the error path", kind)
		}
	}
}

func TestNewUsageBus_NATSFailsClosedWhenUnreachable(t *testing.T) {
	quiet := slog.New(slog.DiscardHandler)
	cfg := config.EventbusConfig{Enabled: true, Kind: "nats", URL: "nats://127.0.0.1:1"}
	bus, err := newUsageBus(context.Background(), &cfg, quiet)
	if err == nil {
		bus.Close()
		t.Fatal("an unreachable nats server must not yield a usable bus")
	}
	if bus.EventBus != nil {
		t.Fatal("failed nats construction must not leak a bus")
	}
}

func TestNewUsageBus_NATSRequiresURL(t *testing.T) {
	quiet := slog.New(slog.DiscardHandler)
	cfg := config.EventbusConfig{Enabled: true, Kind: "nats"}
	if _, err := newUsageBus(context.Background(), &cfg, quiet); err == nil {
		t.Fatal("nats without a URL must be rejected")
	}
}

func TestNewUsageBus_BuildsInMemory(t *testing.T) {
	quiet := slog.New(slog.DiscardHandler)
	cfg := config.EventbusConfig{Enabled: true, Kind: "inmemory"}
	bus, err := newUsageBus(context.Background(), &cfg, quiet)
	if err != nil {
		t.Fatalf("inmemory must build: %v", err)
	}
	defer bus.Close()
	if bus.EventBus == nil {
		t.Fatal("expected a usable transport")
	}
	if !bus.inProcess {
		t.Fatal("inmemory bus must be flagged in-process so the worker does not double-subscribe")
	}
}
