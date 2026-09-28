package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/domain"
)

// Service provides live activity querying and real-time streaming.
type Service struct {
	client *redis.Client
	logger *slog.Logger
}

// NewService creates a new activity service.
func NewService(client *redis.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		client: client,
		logger: logger,
	}
}

// ActiveRequests returns the currently active requests for an organization.
func (s *Service) ActiveRequests(ctx context.Context, orgID uuid.UUID) ([]domain.ActivityEvent, error) {
	if s.client == nil {
		return nil, nil // No Redis
	}

	key := fmt.Sprintf("gateway:activity:active:%s", orgID.String())
	data, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var events []domain.ActivityEvent
	for _, val := range data {
		var ev domain.ActivityEvent
		if err := json.Unmarshal([]byte(val), &ev); err == nil {
			events = append(events, ev)
		}
	}
	return events, nil
}

// Subscribe returns a channel of live activity events for an organization.
func (s *Service) Subscribe(ctx context.Context, orgID uuid.UUID) (<-chan domain.ActivityEvent, error) {
	if s.client == nil {
		ch := make(chan domain.ActivityEvent)
		close(ch)
		return ch, nil
	}

	channel := fmt.Sprintf("gateway:activity:stream:%s", orgID.String())
	pubsub := s.client.Subscribe(ctx, channel)

	events := make(chan domain.ActivityEvent, 100)

	go func() {
		defer pubsub.Close()
		defer close(events)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-pubsub.Channel():
				if !ok {
					return
				}
				var ev domain.ActivityEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil {
					select {
					case events <- ev:
					default:
						// Drop if slow receiver
					}
				}
			}
		}
	}()

	return events, nil
}
