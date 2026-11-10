package gatewayv2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func settlementGateway(budgets ...BudgetConfig) (*Gateway, *BudgetGuardImpl, PolicySnapshot, GatewayCredential) {
	guard := NewBudgetGuard()
	g := &Gateway{budgetGuard: guard, config: DefaultGatewayConfig()}
	cred := GatewayCredential{ID: uuid.New(), OrganizationID: uuid.New()}
	policy := PolicySnapshot{OrganizationID: cred.OrganizationID, Budgets: map[string]BudgetConfig{}}
	for _, b := range budgets {
		policy.Budgets[b.Key] = b
	}
	return g, guard, policy, cred
}

// A ticket settles once: a release after a charge, or a second charge, must not
// move the counters again.
func TestBudgetTicket_SettlesOnce(t *testing.T) {
	g, guard, policy, cred := settlementGateway(BudgetConfig{
		Key: "org:x:tokens", Limit: 1000, Window: time.Hour, Scope: "organization", LimitType: "tokens",
	})
	hints := InspectorHints{EstimatedInputTokens: 100, EstimatedOutputTokens: 100}

	ticket, _, ok := g.admitBudgets(context.Background(), policy, cred, hints, "req-1")
	if !ok {
		t.Fatal("admission refused")
	}
	ticket.Charge(0.5, 300)
	ticket.Release(errors.New("request terminated"))
	ticket.Charge(0.5, 300)

	spent, reserved, _, _ := guard.GetBudgetState("org:x:tokens")
	if spent != 300 || reserved != 0 {
		t.Fatalf("want 300 spent and nothing reserved, got spent=%v reserved=%v", spent, reserved)
	}
}

// When a later budget refuses, the reservations already taken are returned.
func TestAdmitBudgets_RollsBackOnRefusal(t *testing.T) {
	g, guard, policy, cred := settlementGateway(
		BudgetConfig{Key: "org:x:spend", Limit: 100, Window: time.Hour, Scope: "organization", LimitType: "spend"},
		BudgetConfig{Key: "org:x:requests", Limit: 1, Window: time.Hour, Scope: "organization", LimitType: "requests"},
	)
	hints := InspectorHints{EstimatedInputTokens: 1000}

	first, _, ok := g.admitBudgets(context.Background(), policy, cred, hints, "req-1")
	if !ok {
		t.Fatal("first request should be admitted")
	}
	if _, _, ok := g.admitBudgets(context.Background(), policy, cred, hints, "req-2"); ok {
		t.Fatal("second request should hit the one-request cap")
	}
	first.Release(errors.New("done"))

	if _, reserved, _, _ := guard.GetBudgetState("org:x:spend"); reserved != 0 {
		t.Fatalf("refused request left %v reserved on the spend budget", reserved)
	}
}

// Without any configured budget the organization is held to the default limit
// under one org-scoped key, whatever the credential's team or subject.
func TestAdmitBudgets_DefaultIsOrgScoped(t *testing.T) {
	g, guard, policy, cred := settlementGateway()
	team := uuid.New()
	cred.TeamID = &team

	ticket, _, ok := g.admitBudgets(context.Background(), policy, cred, InspectorHints{EstimatedInputTokens: 1000}, "req-1")
	if !ok {
		t.Fatal("admission refused")
	}
	defer ticket.Release(nil)

	if _, reserved, limit, ok := guard.GetBudgetState("org:" + cred.OrganizationID.String()); !ok || reserved == 0 || limit != g.config.DefaultBudgetLimit {
		t.Fatalf("default budget not held on the org key: ok=%v reserved=%v limit=%v", ok, reserved, limit)
	}
}
