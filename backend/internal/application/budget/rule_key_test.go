package budget

import (
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

func TestIsBudgetRule(t *testing.T) {
	orgID, teamID := uuid.New(), uuid.New()
	team := domain.Budget{ID: uuid.New(), Scope: domain.BudgetScopeTeam, ScopeID: teamID}
	org := domain.Budget{ID: uuid.New(), Scope: domain.BudgetScopeOrganization}

	teamRule := domain.PolicyRule{Conditions: []byte(`{"key":"team:` + teamID.String() + `"}`)}
	if !isBudgetRule(teamRule, team, orgID) {
		t.Error("a budget must find its own rule, or every update adds a duplicate")
	}
	if isBudgetRule(teamRule, org, orgID) {
		t.Error("the organization budget must not claim a team budget's rule")
	}

	orgRule := domain.PolicyRule{Conditions: []byte(`{"key":"org:` + orgID.String() + `"}`)}
	if !isBudgetRule(orgRule, org, orgID) {
		t.Error("the organization budget must find its own rule")
	}
}
