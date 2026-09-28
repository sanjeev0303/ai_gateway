package bootstrap

import (
	"testing"

	"ai-governance-saas/backend/internal/domain"
)

// TestProviderAdapters pins the exact key set. azure_openai and custom are
// valid domain kinds deliberately left unmapped: pointing them at another
// upstream's client would validate a credential against the wrong provider.
func TestProviderAdapters(t *testing.T) {
	got := providerAdapters()

	want := map[string]bool{
		domain.ProviderKindAnthropic: true,
		domain.ProviderKindOpenAI:    true,
		domain.ProviderKindGemini:    true,
	}

	if len(got) != len(want) {
		t.Errorf("providerAdapters() has %d kinds, want %d", len(got), len(want))
	}
	for kind := range want {
		adapter, ok := got[kind]
		switch {
		case !ok:
			t.Errorf("providerAdapters() missing kind %q", kind)
		case adapter == nil:
			t.Errorf("providerAdapters()[%q] is nil; credential create would panic", kind)
		}
	}
	for kind := range got {
		if !want[kind] {
			t.Errorf("providerAdapters() has unexpected kind %q", kind)
		}
	}
}
