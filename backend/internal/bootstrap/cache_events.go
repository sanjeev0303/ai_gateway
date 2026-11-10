package bootstrap

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/infrastructure/invalidation"
	"ai-governance-saas/backend/internal/interfaces/gatewayv2"
)

// redisCacheEvents turns the control plane's Redis invalidation messages into
// gateway cache events. It only decodes; the gateway decides what each event
// evicts. The channel closes when ctx ends or the subscription drops.
func redisCacheEvents(ctx context.Context, client *redis.Client) <-chan gatewayv2.CacheEvent {
	out := make(chan gatewayv2.CacheEvent)
	pubsub := client.Subscribe(ctx, invalidation.PolicyInvalidationChannel, invalidation.CredentialInvalidationChannel)

	go func() {
		defer close(out)
		defer pubsub.Close()
		messages := pubsub.Channel()
		for {
			var msg *redis.Message
			select {
			case <-ctx.Done():
				return
			case m, ok := <-messages:
				if !ok {
					return
				}
				msg = m
			}
			ev, ok := decodeCacheEvent(msg.Channel, msg.Payload)
			if !ok {
				continue
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func decodeCacheEvent(channel, payload string) (gatewayv2.CacheEvent, bool) {
	switch channel {
	case invalidation.PolicyInvalidationChannel:
		var p invalidation.PolicyInvalidationPayload
		if json.Unmarshal([]byte(payload), &p) != nil {
			return gatewayv2.CacheEvent{}, false
		}
		orgID, err := uuid.Parse(p.OrganizationID)
		if err != nil {
			return gatewayv2.CacheEvent{}, false
		}
		return gatewayv2.CacheEvent{OrganizationID: orgID}, true
	case invalidation.CredentialInvalidationChannel:
		var p invalidation.CredentialInvalidationPayload
		if json.Unmarshal([]byte(payload), &p) != nil || p.CredentialID == "" {
			return gatewayv2.CacheEvent{}, false
		}
		orgID, _ := uuid.Parse(p.OrganizationID)
		return gatewayv2.CacheEvent{OrganizationID: orgID, CredentialID: p.CredentialID}, true
	}
	return gatewayv2.CacheEvent{}, false
}
