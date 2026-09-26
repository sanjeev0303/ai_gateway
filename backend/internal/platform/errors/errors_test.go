package errors

import (
	stderrors "errors"
	"testing"
)

func TestKindOfWalksChain(t *testing.T) {
	base := New(KindNotFound, "org_not_found", "organization not found")
	if got := KindOf(base); got != KindNotFound {
		t.Errorf("KindOf(base) = %v, want NotFound", got)
	}

	wrapped := Wrap(KindConflict, "conflict", "already exists", base)
	if got := KindOf(wrapped); got != KindConflict {
		t.Errorf("KindOf(wrapped) = %v, want Conflict", got)
	}
}

func TestKindOfDefaultsToInternal(t *testing.T) {
	if got := KindOf(stderrors.New("boom")); got != KindInternal {
		t.Errorf("KindOf(plain error) = %v, want Internal", got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := stderrors.New("db down")
	e := Wrap(KindUnavailable, "store_unavailable", "store unavailable", cause)
	if !stderrors.Is(e, cause) {
		t.Error("errors.Is(e, cause) = false, want true")
	}
	if got := KindOf(e); got != KindUnavailable {
		t.Errorf("KindOf(e) = %v, want Unavailable", got)
	}
}

func TestCodeOrKind(t *testing.T) {
	e := New(KindInvalidArgument, "bad_field", "message")
	if e.CodeOrKind() != "bad_field" {
		t.Errorf("CodeOrKind() = %q, want bad_field", e.CodeOrKind())
	}
	noCode := New(KindInvalidArgument, "", "message")
	if noCode.CodeOrKind() != "invalid_argument" {
		t.Errorf("CodeOrKind() = %q, want invalid_argument", noCode.CodeOrKind())
	}
}

func TestErrorErrorFallback(t *testing.T) {
	if got := New(KindInternal, "", "").Error(); got == "" {
		t.Error("Error() returned empty string")
	}
}
