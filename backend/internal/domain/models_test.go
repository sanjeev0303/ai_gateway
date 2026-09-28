package domain

import "testing"

// Model catalog metadata and provider health state: the governance-owned
// catalog types introduced with M4 model registry seeding.

func TestProviderHealthStateHealthy(t *testing.T) {
	cases := []struct {
		name   string
		state  ProviderHealthState
		expect bool
	}{
		{"healthy admits traffic", ProviderHealthState{Status: ProviderHealthHealthy}, true},
		{"degraded admits traffic", ProviderHealthState{Status: ProviderHealthDegraded}, true},
		{"unhealthy blocks traffic", ProviderHealthState{Status: ProviderHealthUnhealthy}, false},
		{"unknown blocks traffic", ProviderHealthState{Status: ProviderHealthUnknown}, false},
		{"zero state blocks traffic", ProviderHealthState{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.Healthy(); got != tc.expect {
				t.Fatalf("Healthy() = %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestModelStatusConstants(t *testing.T) {
	if ModelStatusActive != "active" || ModelStatusInactive != "inactive" {
		t.Fatalf("model status constants drifted: active=%q inactive=%q",
			ModelStatusActive, ModelStatusInactive)
	}
	if ModelStatusActive == ModelStatusInactive {
		t.Fatal("active and inactive statuses must differ")
	}
}

func TestProviderHealthConstants(t *testing.T) {
	distinct := map[string]bool{}
	for _, s := range []string{
		ProviderHealthUnknown, ProviderHealthHealthy,
		ProviderHealthDegraded, ProviderHealthUnhealthy,
	} {
		if s == "" {
			t.Fatal("provider health constant must not be empty")
		}
		if distinct[s] {
			t.Fatalf("provider health constants must be distinct, got duplicate %q", s)
		}
		distinct[s] = true
	}
}

func TestModelMetadataCarriesGovernanceFields(t *testing.T) {
	md := ModelMetadata{
		ProviderKind:      ProviderKindGemini,
		ModelSlug:         "gemini-2.5-flash",
		DisplayName:       "Gemini 2.5 Flash",
		Capabilities:      []string{"generateContent"},
		ContextLimit:      1000000,
		SupportsStreaming: true,
		SupportsTools:     true,
		SupportsReasoning: false,
		Status:            ModelStatusActive,
		PricingVersion:    "pv-2026-01",
	}
	if md.ProviderKind != ProviderKindGemini {
		t.Fatalf("ProviderKind = %q, want %q", md.ProviderKind, ProviderKindGemini)
	}
	if md.ModelSlug != "gemini-2.5-flash" {
		t.Fatalf("ModelSlug = %q", md.ModelSlug)
	}
	if md.Status != ModelStatusActive {
		t.Fatalf("Status = %q, want %q", md.Status, ModelStatusActive)
	}
	if md.PricingVersion != "pv-2026-01" {
		t.Fatalf("PricingVersion = %q", md.PricingVersion)
	}
	if !md.SupportsStreaming || !md.SupportsTools || md.SupportsReasoning {
		t.Fatal("feature gates not carried through as seeded")
	}
	if md.ContextLimit != 1000000 {
		t.Fatalf("ContextLimit = %d, want 1000000", md.ContextLimit)
	}
}
