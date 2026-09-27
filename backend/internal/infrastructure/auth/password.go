package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ai-governance-saas/backend/internal/ports"
)

// DefaultPasswordIterations is the PBKDF2 cost applied to newly created
// passwords. It is stored inside every digest, so raising it later only
// affects new hashes; existing ones keep verifying at their original cost
// until the next successful sign-in upgrades them.
const DefaultPasswordIterations = 600_000

const (
	passwordAlgorithm   = "pbkdf2-sha256"
	passwordSaltLen     = 16
	passwordKeyLen      = 32
	minPasswordCost     = 10_000
	maxPasswordCost     = 5_000_000
	passwordHashParts   = 5
	decoyPasswordLength = 32
)

// ErrPasswordMismatch is the single failure returned for a wrong password, an
// unknown account, an unparsable stored digest, and an unsupported algorithm.
// Collapsing them here is what stops a sign-in endpoint from turning into an
// oracle that reports which emails are registered.
var ErrPasswordMismatch = errors.New("password verification failed")

// pbkdf2Hasher derives and verifies password digests with PBKDF2-HMAC-SHA256
// from the standard library, so password storage needs no third-party
// dependency.
type pbkdf2Hasher struct {
	iterations int
}

// NewPBKDF2PasswordHasher returns a PasswordHasher. Passing a non-positive
// iteration count selects DefaultPasswordIterations; tests pass a low count
// to stay fast, and verification still honours the cost recorded in the
// stored digest rather than this value.
func NewPBKDF2PasswordHasher(iterations int) ports.PasswordHasher {
	if iterations <= 0 {
		iterations = DefaultPasswordIterations
	}
	return pbkdf2Hasher{iterations: iterations}
}

func (h pbkdf2Hasher) Hash(_ context.Context, password string) (string, error) {
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, h.iterations, passwordKeyLen)
	if err != nil {
		return "", fmt.Errorf("derive password key: %w", err)
	}
	return fmt.Sprintf("$%s$%d$%s$%s",
		passwordAlgorithm,
		h.iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h pbkdf2Hasher) Verify(_ context.Context, encoded, password string) error {
	algorithm, iterations, salt, want, err := parsePasswordDigest(encoded)
	if err != nil || algorithm != passwordAlgorithm {
		return ErrPasswordMismatch
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return ErrPasswordMismatch
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

// DecoyPassword is a fixed, meaningless string. Verifying a submitted password
// against a digest of it burns the same CPU a real check would, which is what
// keeps sign-in timing from revealing whether an email is registered.
func DecoyPassword() string {
	return strings.Repeat("d", decoyPasswordLength)
}

// decoyPasswordDigest is the digest of a random 32-byte secret nobody holds,
// precomputed so the equalizer does not cost 600k iterations per request.
const decoyPasswordDigest = "$pbkdf2-sha256$600000$ip+mRenmZV006B1AcJuglg$kht2Q46t9EMGXwHClKpVfiWcFRIHzvMaSYfi1Frr7/k"

// DecoyPasswordDigest returns the digest to verify a submitted password
// against when there is no real one to check, so an unknown email costs the
// same as a known one.
func DecoyPasswordDigest() string {
	return decoyPasswordDigest
}

func (h pbkdf2Hasher) DecoyDigest() string {
	return decoyPasswordDigest
}

func parsePasswordDigest(encoded string) (string, int, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != passwordHashParts || parts[0] != "" {
		return "", 0, nil, nil, errors.New("malformed password digest")
	}
	iterations, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, nil, nil, errors.New("malformed password digest: cost")
	}
	// A tampered row must not be able to turn one sign-in into a CPU
	// exhaustion denial of service.
	if iterations < minPasswordCost || iterations > maxPasswordCost {
		return "", 0, nil, nil, errors.New("password digest cost out of range")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return "", 0, nil, nil, errors.New("malformed password digest: salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return "", 0, nil, nil, errors.New("malformed password digest: key")
	}
	if len(salt) == 0 || len(key) == 0 {
		return "", 0, nil, nil, errors.New("malformed password digest: empty component")
	}
	return parts[1], iterations, salt, key, nil
}
