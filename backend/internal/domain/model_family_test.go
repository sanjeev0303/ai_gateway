package domain

import "testing"

func TestProviderForModel(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4-5":                     ProviderKindAnthropic,
		"gpt-4o-mini":                           ProviderKindOpenAI,
		"o3-mini":                               ProviderKindOpenAI,
		"text-embedding-3-small":                ProviderKindOpenAI,
		"gemini-2.5-flash":                      ProviderKindGemini,
		"meta-llama/Llama-3.3-70B-Instruct":     ProviderKindHuggingFace,
		"mistralai/Mistral-7B-Instruct-v0.3":    ProviderKindHuggingFace,
		"hf-mistralai/Mistral-7B-Instruct-v0.3": ProviderKindHuggingFace,
		"mistral-large-latest":                  ProviderKindCustom,
		"open-mistral-nemo":                     ProviderKindCustom,
		"codestral-2501":                        ProviderKindCustom,
		"voxtral-mini-latest":                   ProviderKindCustom,
		"pixtral-large-latest":                  ProviderKindCustom,
		"devstral-small-2505":                   ProviderKindCustom,
		"gemma-4-31b-it":                        "",
		"":                                      "",
	}
	for model, want := range cases {
		if got := ProviderForModel(model); got != want {
			t.Errorf("ProviderForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

// The model list the pricing page falls back to must quote the same rates the
// gateway bills with, or the two drift apart again.
func TestCanonicalPricingSpecsMatchBillingLookup(t *testing.T) {
	for _, spec := range CanonicalPricingSpecs {
		billed, ok := LookupCanonicalPricing(spec.ProviderKind, spec.ModelSlug)
		if !ok {
			t.Errorf("%s: no billing price", spec.ModelSlug)
			continue
		}
		if billed.InputRatePerM != spec.InputRatePerM || billed.OutputRatePerM != spec.OutputRatePerM || billed.CachedReadRatePerM != spec.CachedReadRatePerM {
			t.Errorf("%s: listed %+v, billed %+v", spec.ModelSlug, spec, billed)
		}
	}
}
