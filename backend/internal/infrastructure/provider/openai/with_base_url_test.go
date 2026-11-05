package openai

import "testing"

func TestWithBaseURL(t *testing.T) {
	base := NewCustomProviderAdapter("https://api.mistral.ai", "", true)
	cases := map[string]string{
		"https://api.mistral.ai/v1":  "https://api.mistral.ai",
		"https://api.mistral.ai/v1/": "https://api.mistral.ai",
		"http://vllm.internal:8000":  "http://vllm.internal:8000",
		" https://llm.example.com/ ": "https://llm.example.com",
	}
	for in, want := range cases {
		got, ok := base.WithBaseURL(in).(*ProviderAdapter)
		if !ok || got.baseURL != want {
			t.Errorf("WithBaseURL(%q) = %+v, want baseURL %q", in, got, want)
		}
	}
	if base.WithBaseURL("") != base {
		t.Error("empty base URL must keep the original adapter")
	}
	if base.baseURL != "https://api.mistral.ai" {
		t.Error("WithBaseURL must not change the receiver")
	}
}
