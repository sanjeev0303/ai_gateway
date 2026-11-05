package controlapi

import (
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

func TestModelAccessEntriesForReadsBothRuleFormats(t *testing.T) {
	pol := domain.Policy{ID: uuid.New(), Priority: 100}

	fromPoliciesPage := domain.PolicyRule{
		ID: uuid.New(), RuleType: domain.RuleTypeBlocklist, Enabled: true,
		Conditions: []byte(`{}`),
		Parameters: []byte(`{"models":["gemini-2.5-pro","gemini-3.1-pro-preview"]}`),
	}
	got := modelAccessEntriesFor(pol, fromPoliciesPage)
	if len(got) != 2 || got[0].Provider != anyProvider || got[0].Model != "gemini-2.5-pro" || got[0].Allowed {
		t.Fatalf("policies-page rule entries = %+v", got)
	}

	fromModelAccessPage := domain.PolicyRule{
		ID: uuid.New(), RuleType: domain.RuleTypeAllowlist, Enabled: true,
		Conditions: []byte(`{"provider":"gemini","model":"gemini-2.5-flash"}`),
		Parameters: []byte(`{}`),
	}
	got = modelAccessEntriesFor(pol, fromModelAccessPage)
	if len(got) != 1 || got[0].Provider != "gemini" || got[0].Model != "gemini-2.5-flash" || !got[0].Allowed {
		t.Fatalf("model-access-page rule entries = %+v", got)
	}

	rateLimit := domain.PolicyRule{RuleType: domain.RuleTypeRateLimit, Parameters: []byte(`{"limit":3}`)}
	if got := modelAccessEntriesFor(pol, rateLimit); len(got) != 0 {
		t.Fatalf("rate limit rule produced model access entries: %+v", got)
	}
}
