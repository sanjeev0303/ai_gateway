package domain

import "testing"

func TestPrivacyModeValid(t *testing.T) {
	for _, m := range []PrivacyMode{PrivacyOff, PrivacyMetadataOnly, PrivacyPromptOnly, PrivacyFullContent} {
		if !m.Valid() {
			t.Errorf("Valid(%q) = false, want true", m)
		}
	}
	for _, m := range []PrivacyMode{"", "off", "metadata_only", "PROMPTS", "FULL"} {
		if m.Valid() {
			t.Errorf("Valid(%q) = true, want false", m)
		}
	}
}

func TestParsePrivacyModeRejectsUnknown(t *testing.T) {
	if _, ok := ParsePrivacyMode("EVERYTHING"); ok {
		t.Error("ParsePrivacyMode(\"EVERYTHING\") accepted an unsupported mode")
	}
	m, ok := ParsePrivacyMode(string(PrivacyPromptOnly))
	if !ok || m != PrivacyPromptOnly {
		t.Errorf("ParsePrivacyMode = (%q, %v), want (%q, true)", m, ok, PrivacyPromptOnly)
	}
}

func TestResolvePrivacyMode(t *testing.T) {
	override := func(s string) *string { return &s }

	cases := []struct {
		name     string
		override *string
		fallback PrivacyMode
		want     PrivacyMode
	}{
		{"inherits valid default", nil, PrivacyFullContent, PrivacyFullContent},
		{"override wins over default", override("PROMPT_ONLY"), PrivacyFullContent, PrivacyPromptOnly},
		{"invalid default fails closed", nil, "GARBAGE", PrivacyMetadataOnly},
		{"empty default fails closed", nil, "", PrivacyMetadataOnly},
		{"invalid override fails closed", override("ALL"), PrivacyFullContent, PrivacyMetadataOnly},
		{"blank override fails closed", override(""), PrivacyFullContent, PrivacyMetadataOnly},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolvePrivacyMode(tc.override, tc.fallback); got != tc.want {
				t.Errorf("ResolvePrivacyMode = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestContentVisible(t *testing.T) {
	roles := []string{"system", "user", "assistant", "tool"}
	visible := map[PrivacyMode]map[string]bool{
		PrivacyOff:          {"system": true, "user": true, "assistant": true, "tool": true},
		PrivacyFullContent:  {"system": true, "user": true, "assistant": true, "tool": true},
		PrivacyPromptOnly:   {"system": true, "user": true, "assistant": false, "tool": false},
		PrivacyMetadataOnly: {"system": false, "user": false, "assistant": false, "tool": false},
		"UNRECOGNISED":      {"system": false, "user": false, "assistant": false, "tool": false},
	}
	for mode, want := range visible {
		for _, role := range roles {
			if got := mode.ContentVisible(role); got != want[role] {
				t.Errorf("PrivacyMode(%q).ContentVisible(%q) = %v, want %v", mode, role, got, want[role])
			}
		}
	}
}

func TestRedacted(t *testing.T) {
	msg := ConversationMessage{Role: "assistant", Content: "sensitive output"}

	for _, mode := range []PrivacyMode{PrivacyMetadataOnly, PrivacyPromptOnly, "UNRECOGNISED"} {
		if got := msg.Redacted(mode); got.Content != "" {
			t.Errorf("Redacted(%q) kept assistant content %q", mode, got.Content)
		}
	}
	for _, mode := range []PrivacyMode{PrivacyOff, PrivacyFullContent} {
		if got := msg.Redacted(mode); got.Content != msg.Content {
			t.Errorf("Redacted(%q) dropped content that the mode permits", mode)
		}
	}

	user := ConversationMessage{Role: "user", Content: "my prompt"}
	if got := user.Redacted(PrivacyPromptOnly); got.Content != user.Content {
		t.Error("Redacted(PROMPT_ONLY) dropped user content, which the mode permits")
	}
}

// TestRedactedDoesNotMutateReceiver guards the copy semantics the service
// relies on: redacting one response must not blank the stored message.
func TestRedactedDoesNotMutateReceiver(t *testing.T) {
	original := ConversationMessage{Role: "assistant", Content: "keep me"}
	_ = original.Redacted(PrivacyMetadataOnly)
	if original.Content != "keep me" {
		t.Errorf("Redacted mutated the receiver: content is now %q", original.Content)
	}
}
