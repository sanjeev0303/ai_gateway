package gatewayv2

import (
	"context"
	"errors"
	"sync"
)

// budgetHold is one budget reservation and the unit that budget counts in.
type budgetHold struct {
	release   BudgetReleaseFunc
	limitType string
}

// budgetTicket is a request's claim on every budget that applies to it. It
// settles exactly once: either Charge bills the finished usage or Release gives
// the reservations back. Later calls do nothing, so the pipeline can release on
// every exit path without double counting.
type budgetTicket struct {
	holds []budgetHold
	once  sync.Once
}

// Charge bills finished usage to every held budget in that budget's own unit:
// dollars, tokens or one request.
func (t *budgetTicket) Charge(cost float64, tokens int) {
	if t == nil {
		return
	}
	t.once.Do(func() {
		for _, h := range t.holds {
			h.release(budgetUnits(h.limitType, cost, tokens), nil)
		}
	})
}

// Release gives every reservation back without charging anything.
func (t *budgetTicket) Release(reason error) {
	if t == nil {
		return
	}
	t.once.Do(func() {
		for _, h := range t.holds {
			h.release(0, reason)
		}
	})
}

// admitBudgets reserves the request's estimated cost against every budget that
// applies to the credential. An organization with no budgets is held to the
// gateway's default limit. When any budget refuses, the reservations already
// taken are given back and the refusing result is returned.
func (g *Gateway) admitBudgets(ctx context.Context, policy PolicySnapshot, cred GatewayCredential, hints InspectorHints, reqID string) (*budgetTicket, BudgetResult, bool) {
	g.budgetGuard.SyncFromSnapshot(policy)

	budgets := policy.applicableBudgets(cred)
	if len(budgets) == 0 {
		budgets = []BudgetConfig{{
			Key:       "org:" + cred.OrganizationID.String(),
			Limit:     g.config.DefaultBudgetLimit,
			Window:    g.config.DefaultBudgetWindow,
			LimitType: "spend",
		}}
	}

	cost := estimateCost(hints)
	tokens := hints.EstimatedInputTokens + hints.EstimatedOutputTokens
	if tokens <= 0 {
		tokens = 1
	}

	ticket := &budgetTicket{}
	var last BudgetResult
	for _, cfg := range budgets {
		result, release := g.budgetGuard.Reserve(ctx, cfg.Key, cfg.Limit, cfg.Window, budgetUnits(cfg.LimitType, cost, tokens), reqID)
		if !result.Allowed {
			ticket.Release(errors.New("budget_exceeded"))
			return nil, result, false
		}
		ticket.holds = append(ticket.holds, budgetHold{release: release, limitType: cfg.LimitType})
		last = result
	}
	return ticket, last, true
}

// budgetUnits converts a request into what a budget counts. A request budget
// counts the request itself; charging it dollars let a cap of 3 requests
// behave like $3.
func budgetUnits(limitType string, cost float64, tokens int) float64 {
	switch limitType {
	case "tokens":
		return float64(tokens)
	case "requests":
		return 1
	}
	return cost
}
