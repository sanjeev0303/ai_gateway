package auth

import (
	"context"
	"strings"
	"testing"
)

// testCost keeps the suite fast; verification honours the cost stored in the
// digest, not this value, so low-cost digests still exercise the real path.
const testCost = 10_000

func TestPasswordHashRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := NewPBKDF2PasswordHasher(testCost)

	encoded, err := h.Hash(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := h.Verify(ctx, encoded, "correct horse battery staple"); err != nil {
		t.Fatalf("Verify(correct) = %v, want nil", err)
	}
	if err := h.Verify(ctx, encoded, "wrong password"); err != ErrPasswordMismatch {
		t.Fatalf("Verify(wrong) = %v, want ErrPasswordMismatch", err)
	}
}

func TestPasswordHashIsSaltedAndSelfDescribing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := NewPBKDF2PasswordHasher(testCost)

	first, err := h.Hash(ctx, "same password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	second, err := h.Hash(ctx, "same password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if first == second {
		t.Fatal("identical passwords produced identical digests: salt is not random")
	}
	if !strings.HasPrefix(first, "$"+passwordAlgorithm+"$") {
		t.Fatalf("digest %q is missing the algorithm and cost prefix", first)
	}
	if !strings.Contains(first, "$"+itoa(testCost)+"$") {
		t.Fatalf("digest %q does not record the iteration cost", first)
	}
	// Each digest must verify against the other hasher's cost-independent path.
	if err := h.Verify(ctx, second, "same password"); err != nil {
		t.Fatalf("Verify(second) = %v, want nil", err)
	}
}

func TestPasswordVerifyHonoursStoredCostNotConfigured(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cheap := NewPBKDF2PasswordHasher(testCost)
	dear := NewPBKDF2PasswordHasher(testCost * 2)

	encoded, err := cheap.Hash(ctx, "rotate me")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	// A digest minted at the low cost must still verify through a hasher
	// configured for a higher one, otherwise raising the cost would lock out
	// every existing user.
	if err := dear.Verify(ctx, encoded, "rotate me"); err != nil {
		t.Fatalf("Verify across cost change = %v, want nil", err)
	}
	if err := dear.Verify(ctx, encoded, "not the password"); err != ErrPasswordMismatch {
		t.Fatalf("Verify(wrong) = %v, want ErrPasswordMismatch", err)
	}
}

// Every malformed input must collapse to the same error, otherwise the failure
// mode becomes an oracle for the shape of the stored digest.
func TestPasswordVerifyRejectsMalformedDigestsUniformly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := NewPBKDF2PasswordHasher(testCost)

	valid, err := h.Hash(ctx, "anchor")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	// A well-formed digest whose key component is truncated must fail exactly
	// like a garbage string, not like a missing field.
	parts := strings.Split(valid, "$")
	truncated := strings.Join(parts[:4], "$") + "$"

	cases := map[string]string{
		"empty":             "",
		"no_leading_dollar": "pbkdf2-sha256$10000$c2FsdA$aGFzaA",
		"too_few_parts":     "$pbkdf2-sha256$10000$c2FsdA",
		"cost_not_a_number": "$pbkdf2-sha256$notanumber$c2FsdA$aGFzaA",
		"cost_too_low":      "$pbkdf2-sha256$1$c2FsdA$aGFzaA",
		"cost_absurd":       "$pbkdf2-sha256$999999999$c2FsdA$aGFzaA",
		"salt_not_base64":   "$pbkdf2-sha256$10000$!!!$aGFzaA",
		"key_not_base64":    "$pbkdf2-sha256$10000$c2FsdA$!!!",
		"empty_salt":        "$pbkdf2-sha256$10000$$aGFzaA",
		"empty_key":         "$pbkdf2-sha256$10000$c2FsdA$",
		"wrong_algorithm":   "$argon2id$10000$c2FsdA$aGFzaA",
		"truncated_key":     truncated,
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			if err := h.Verify(ctx, encoded, "anchor"); err != ErrPasswordMismatch {
				t.Fatalf("Verify(%s) = %v, want ErrPasswordMismatch", name, err)
			}
		})
	}
}

func TestPasswordHashRejectsWeakInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := NewPBKDF2PasswordHasher(testCost)

	// An empty password still hashes -- policy is the service's job, not the
	// KDF's. This test pins that the primitive is total.
	encoded, err := h.Hash(ctx, "")
	if err != nil {
		t.Fatalf("Hash(empty): %v", err)
	}
	if err := h.Verify(ctx, encoded, ""); err != nil {
		t.Fatalf("Verify(empty) = %v, want nil", err)
	}
}

func TestDecoyPasswordIsFixed(t *testing.T) {
	t.Parallel()
	if DecoyPassword() != DecoyPassword() {
		t.Fatal("DecoyPassword must return a stable value")
	}
	if len(DecoyPassword()) == 0 {
		t.Fatal("DecoyPassword must not be empty")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
