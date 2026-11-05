package gatewayv2

import "testing"

func TestBudgetUnits(t *testing.T) {
	if got := budgetUnits("requests", 0.004, 900); got != 1 {
		t.Errorf("request budget charged %v, want 1", got)
	}
	if got := budgetUnits("tokens", 0.004, 900); got != 900 {
		t.Errorf("token budget charged %v, want 900", got)
	}
	if got := budgetUnits("spend", 0.004, 900); got != 0.004 {
		t.Errorf("spend budget charged %v, want 0.004", got)
	}
}
