package domain

import "testing"

func TestFallbackPolicyValidate(t *testing.T) {
	valid := FallbackPolicy{FallbackProvider: ProviderKindAnthropic, FallbackModel: "claude-sonnet-4-5"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	disabled := FallbackPolicy{}
	if err := disabled.Validate(); err != nil {
		t.Fatalf("zero FallbackPolicy Validate() = %v, want nil (fallback disabled)", err)
	}

	unknownProvider := FallbackPolicy{FallbackProvider: "not-a-provider", FallbackModel: "claude-sonnet-4-5"}
	if err := unknownProvider.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for unknown fallback provider")
	}

	emptyModel := FallbackPolicy{FallbackProvider: ProviderKindAnthropic}
	if err := emptyModel.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for missing fallback model")
	}
}

func TestIsProviderKind(t *testing.T) {
	for _, kind := range []string{ProviderKindOpenAI, ProviderKindAnthropic, ProviderKindGemini, ProviderKindAzureOpenAI, ProviderKindCustom} {
		if !IsProviderKind(kind) {
			t.Fatalf("IsProviderKind(%q) = false, want true", kind)
		}
	}
	for _, kind := range []string{"", "not-a-provider", "claude"} {
		if IsProviderKind(kind) {
			t.Fatalf("IsProviderKind(%q) = true, want false", kind)
		}
	}
}