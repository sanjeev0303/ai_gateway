package gatewayv2

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/domain"
)

// ActivityTracker tracks real-time gateway activity without blocking requests.
type ActivityTracker struct {
	client *redis.Client
	logger *slog.Logger
	queue  chan domain.ActivityEvent
	wg     sync.WaitGroup
	stopCh chan struct{}
	drops  int64
}

// NewActivityTracker creates an activity tracker connecting to Redis.
func NewActivityTracker(client *redis.Client, logger *slog.Logger) *ActivityTracker {
	if logger == nil {
		logger = slog.Default()
	}
	t := &ActivityTracker{
		client: client,
		logger: logger,
		queue:  make(chan domain.ActivityEvent, 10000),
		stopCh: make(chan struct{}),
	}

	if client != nil {
		t.wg.Add(1)
		go t.worker()
	}
	return t
}

// Emit queues an activity event for processing.
func (t *ActivityTracker) Emit(event domain.ActivityEvent) {
	if t.client == nil {
		return
	}
	select {
	case t.queue <- event:
	default:
		atomic.AddInt64(&t.drops, 1)
	}
}

// Close gracefully stops the tracker.
func (t *ActivityTracker) Close() {
	if t.client != nil {
		close(t.stopCh)
		t.wg.Wait()
	}
}

func (t *ActivityTracker) worker() {
	defer t.wg.Done()

	for {
		select {
		case <-t.stopCh:
			// Drain
			for {
				select {
				case ev := <-t.queue:
					t.process(ev)
				default:
					return
				}
			}
		case ev := <-t.queue:
			t.process(ev)
		}
	}
}

func (t *ActivityTracker) process(ev domain.ActivityEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	orgKey := fmt.Sprintf("gateway:activity:active:%s", ev.OrganizationID.String())
	channel := fmt.Sprintf("gateway:activity:stream:%s", ev.OrganizationID.String())

	pipe := t.client.Pipeline()

	if ev.CurrentState == domain.ActivityStateCompleted || ev.CurrentState == domain.ActivityStateFailed {
		pipe.HDel(ctx, orgKey, ev.RequestID)
	} else {
		pipe.HSet(ctx, orgKey, ev.RequestID, data)
		pipe.Expire(ctx, orgKey, 24*time.Hour)
	}

	pipe.Publish(ctx, channel, data)

	if _, err := pipe.Exec(ctx); err != nil {
		t.logger.Warn("activity tracker: failed to update redis", "err", err)
	}
}
