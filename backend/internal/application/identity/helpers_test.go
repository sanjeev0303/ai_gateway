package identity

import (
	"encoding/hex"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
)

// Helper-function unit tests: email normalization, slugging, random tokens,
// and the small predicates the service composes them from. No repositories
// are involved.

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"alice@example.com", "alice@example.com"},
		{"  ALICE@Example.COM  ", "alice@example.com"},
		{"\tBOB@EXAMPLE.ORG\n", "bob@example.org"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeEmail(c.input); got != c.want {
			t.Errorf("normalizeEmail(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Apostle's Guild", "apostle-s-guild"},
		{"  A--B! C  ", "a-b-c"},
		{"123ABC", "123abc"},
		{"a!!b", "a-b"},
		{"", ""},
		{"日本語", ""},
	}
	for _, c := range cases {
		if got := slugify(c.input); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestEmailLocalPart(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"alice@example.com", "alice"},
		{"a.b+c@example.com", "a.b+c"},
		{"no-at-sign", "no-at-sign"},
		{"@example.com", "@example.com"},
	}
	for _, c := range cases {
		if got := emailLocalPart(c.input); got != c.want {
			t.Errorf("emailLocalPart(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestIsJoinRole(t *testing.T) {
	for _, role := range []string{domain.RoleAdmin, domain.RoleMember} {
		if !isJoinRole(role) {
			t.Errorf("isJoinRole(%q) = false, want true", role)
		}
	}
	for _, role := range []string{domain.RoleOwner, domain.RoleAuditor, "", "service"} {
		if isJoinRole(role) {
			t.Errorf("isJoinRole(%q) = true, want false", role)
		}
	}
}

func TestRandomHex(t *testing.T) {
	for _, n := range []int{2, 16, 32} {
		got, err := randomHex(n)
		if err != nil {
			t.Fatalf("randomHex(%d): %v", n, err)
		}
		if len(got) != n*2 {
			t.Errorf("len(randomHex(%d)) = %d, want %d", n, len(got), n*2)
		}
		if _, err := hex.DecodeString(got); err != nil {
			t.Errorf("randomHex(%d) = %q is not hex: %v", n, got, err)
		}
	}
	a, _ := randomHex(16)
	b, _ := randomHex(16)
	if a == b {
		t.Error("two randomHex(16) draws collided")
	}
}

func TestNewServiceDefaults(t *testing.T) {
	defaults := NewService(ServiceDeps{})
	if defaults.log == nil {
		t.Error("NewService with nil Log must default to slog.Default()")
	}
	if defaults.now == nil {
		t.Error("NewService with nil Now must default to time.Now")
	}
	if defaults.inviteTTL != defaultInviteTTL {
		t.Errorf("NewService zero InviteTTL: got %v, want %v", defaults.inviteTTL, defaultInviteTTL)
	}
	negative := NewService(ServiceDeps{InviteTTL: -time.Hour})
	if negative.inviteTTL != defaultInviteTTL {
		t.Errorf("NewService negative InviteTTL: got %v, want %v", negative.inviteTTL, defaultInviteTTL)
	}
	explicit := NewService(ServiceDeps{InviteTTL: testInviteTTL})
	if explicit.inviteTTL != testInviteTTL {
		t.Errorf("NewService explicit InviteTTL: got %v, want %v", explicit.inviteTTL, testInviteTTL)
	}
}
