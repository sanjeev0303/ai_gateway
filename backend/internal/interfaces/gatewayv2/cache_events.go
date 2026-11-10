package gatewayv2

import (
	"context"

	"github.com/google/uuid"
)

// CacheEvent tells the gateway that something it caches changed in the
// control plane. Set OrganizationID alone when an organization's policies,
// budgets or provider connections changed; set CredentialID when one gateway
// key changed or was revoked. Which caches that touches is the gateway's
// business, so callers only describe what changed.
type CacheEvent struct {
	OrganizationID uuid.UUID
	CredentialID   string
}

// Subscribe applies cache events until ctx ends or the stream closes. Any
// source can feed it: Redis pub/sub between processes, or a plain channel in
// the same process or a test.
func (g *Gateway) Subscribe(ctx context.Context, events <-chan CacheEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			g.applyCacheEvent(ctx, ev)
		}
	}
}

func (g *Gateway) applyCacheEvent(ctx context.Context, ev CacheEvent) {
	if ev.CredentialID != "" {
		if g.credentials != nil {
			g.credentials.InvalidateByID(ctx, ev.CredentialID)
		}
		return
	}
	if ev.OrganizationID == uuid.Nil {
		return
	}
	if g.policyResolver != nil {
		g.policyResolver.Invalidate(ev.OrganizationID)
	}
	if g.tenantResolver != nil {
		g.tenantResolver.Invalidate(ev.OrganizationID)
	}
}
