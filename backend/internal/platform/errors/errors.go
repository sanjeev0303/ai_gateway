// Package errors defines the typed error taxonomy shared by every AI
// Governance process. Application code returns *Error values without importing
// any HTTP concern; the transport layer maps kinds to status codes via
// httpx.StatusFor.
package errors

import (
	stderrors "errors"
	"fmt"
)

// Kind classifies an error so transport layers map it consistently.
type Kind int

const (
	KindInvalidArgument Kind = iota
	KindUnauthenticated
	KindPermissionDenied
	KindNotFound
	KindConflict
	KindRateLimited
	KindTimeout
	KindUnavailable
	KindInternal
)

// String returns the stable wire-safe identifier for the kind.
func (k Kind) String() string {
	switch k {
	case KindInvalidArgument:
		return "invalid_argument"
	case KindUnauthenticated:
		return "unauthenticated"
	case KindPermissionDenied:
		return "permission_denied"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindRateLimited:
		return "rate_limited"
	case KindTimeout:
		return "timeout"
	case KindUnavailable:
		return "unavailable"
	default:
		return "internal"
	}
}

// Error is a structured application error.
type Error struct {
	// Kind drives the HTTP status mapping.
	Kind Kind
	// Code is a stable machine-readable code returned to API clients (e.g.
	// "org_not_found"); when empty the kind string is used instead.
	Code string
	// Message is safe for end users: no secrets, no stack traces.
	Message string
	// Err is the underlying cause, kept for logging and unwrapping.
	Err error
}

// New returns a *Error without a cause.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap returns a *Error carrying the underlying cause.
func Wrap(kind Kind, code, message string, err error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: err}
}

// Error implements the error interface using the user-safe message.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s error", e.Kind)
}

// Unwrap exposes the underlying cause for errors.Is/As chains.
func (e *Error) Unwrap() error { return e.Err }

// CodeOrKind returns the stable code, falling back to the kind identifier.
func (e *Error) CodeOrKind() string {
	if e.Code != "" {
		return e.Code
	}
	return e.Kind.String()
}

// KindOf walks the error chain and returns the first Kind it finds,
// defaulting to KindInternal for anything that is not an *Error.
func KindOf(err error) Kind {
	for err != nil {
		if e, ok := err.(*Error); ok {
			return e.Kind
		}
		err = stderrors.Unwrap(err)
	}
	return KindInternal
}
