package domain

import "testing"

func TestMistralModelsUseMistralRates(t *testing.T) {
	cases := []struct {
		provider, model string
		want            int64
	}{
		// 20 input, 167 output tokens at $0.15/M each
		{"custom", "open-mistral-nemo", 28},
		// 20 input at $0.30/M, 191 output at $0.90/M
		{"custom", "codestral-2508", 178},
		{"mistral", "mistral-large-latest", 20*2 + 167*6},
	}
	for _, c := range cases {
		out := int64(167)
		if c.model == "codestral-2508" {
			out = 191
		}
		if got := ResolveAndCalculateModelCost(c.provider, c.model, 20, out, 0); got != c.want {
			t.Errorf("%s/%s cost = %d micros, want %d", c.provider, c.model, got, c.want)
		}
	}

	// A Hugging Face repo id that names Mistral keeps the Hugging Face rate.
	spec, ok := LookupCanonicalPricing("huggingface", "mistralai/Mistral-7B-Instruct-v0.3")
	if !ok || spec.ProviderKind != "huggingface" {
		t.Errorf("HF Mistral repo priced as %q, want huggingface", spec.ProviderKind)
	}
	// An unknown custom model still falls back to the generic rate.
	if _, ok := LookupCanonicalPricing("custom", "some-other-model"); ok {
		t.Error("unknown custom model matched a canonical price")
	}
}
