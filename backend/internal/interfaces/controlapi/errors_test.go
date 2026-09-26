package controlapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-governance-saas/backend/internal/application/identity"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// TestWriteServiceErrorMapsKnownErrors drives the full error-envelope mapping
// table: every identity/repository sentinel plus a wrapped error to prove
// errors.Is matching.
func TestWriteServiceErrorMapsKnownErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{name: "unauthenticated", err: identity.ErrUnauthorized, wantStatus: http.StatusUnauthorized, wantCode: "unauthenticated", wantMsg: "missing or invalid credentials"},
		{name: "invalid invitation", err: identity.ErrInvalidInvitation, wantStatus: http.StatusBadRequest, wantCode: "invalid_invitation", wantMsg: identity.ErrInvalidInvitation.Error()},
		{name: "invitation expired", err: identity.ErrInvitationExpired, wantStatus: http.StatusBadRequest, wantCode: "invitation_expired", wantMsg: identity.ErrInvitationExpired.Error()},
		{name: "invitation email mismatch", err: identity.ErrInvitationEmailMismatch, wantStatus: http.StatusBadRequest, wantCode: "invitation_email_mismatch", wantMsg: identity.ErrInvitationEmailMismatch.Error()},
		{name: "invalid", err: repositories.ErrInvalid, wantStatus: http.StatusBadRequest, wantCode: "invalid_argument", wantMsg: repositories.ErrInvalid.Error()},
		{name: "not found", err: repositories.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "not_found", wantMsg: repositories.ErrNotFound.Error()},
		{name: "conflict", err: repositories.ErrConflict, wantStatus: http.StatusConflict, wantCode: "conflict", wantMsg: repositories.ErrConflict.Error()},
		{name: "forbidden", err: repositories.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: "permission_denied", wantMsg: repositories.ErrForbidden.Error()},
		{name: "wrapped not found", err: fmt.Errorf("wrap: %w", repositories.ErrNotFound), wantStatus: http.StatusNotFound, wantCode: "not_found", wantMsg: repositories.ErrNotFound.Error()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			writeServiceError(rec, req, tc.err)

			requireEqual(t, rec.Code, tc.wantStatus)
			var env errorEnvelope
			decodeBody(t, rec.Result(), &env)
			requireEqual(t, env.Err.Code, tc.wantCode)
			requireEqual(t, env.Err.Message, tc.wantMsg)
		})
	}
}

// TestWriteServiceErrorUnexpectedDegradesToInternal proves unknown errors never
// leak details to clients: they become a generic 500.
func TestWriteServiceErrorUnexpectedDegradesToInternal(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	writeServiceError(rec, req, errors.New("boom"))

	requireEqual(t, rec.Code, http.StatusInternalServerError)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "internal")
	requireEqual(t, env.Err.Message, "internal error")
}

type decodeTarget struct {
	Email string `json:"email"`
}

func TestDecodeJSONAcceptsValidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.c"}`))
	var dst decodeTarget
	requireNoErr(t, decodeJSON(rec, req, &dst))
	requireEqual(t, dst.Email, "a@b.c")
}

func TestDecodeJSONRejectsMalformedBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":`))
	var dst decodeTarget
	err := decodeJSON(rec, req, &dst)
	if err == nil {
		t.Fatal("decodeJSON accepted malformed body")
	}
	writeServiceError(rec, req, err)
	requireEqual(t, rec.Code, http.StatusBadRequest)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "invalid_json")
	requireEqual(t, env.Err.Message, "malformed request body")
}

// TestDecodeJSONRejectsUnknownFields proves DisallowUnknownFields is wired in;
// a typo'd field is a client bug, not silently ignored data.
func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.c","nope":1}`))
	var dst decodeTarget
	err := decodeJSON(rec, req, &dst)
	if err == nil {
		t.Fatal("decodeJSON accepted unknown field")
	}
	writeServiceError(rec, req, err)
	requireEqual(t, rec.Code, http.StatusBadRequest)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "invalid_json")
}

// TestDecodeJSONRejectsOversizedBody proves the 1 MiB MaxBytesReader cap bites
// even when the payload is a single oversized string value.
func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	body := `{"email":"` + strings.Repeat("a", 1<<20+1) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var dst decodeTarget
	err := decodeJSON(rec, req, &dst)
	if err == nil {
		t.Fatal("decodeJSON accepted oversized body")
	}
	writeServiceError(rec, req, err)
	requireEqual(t, rec.Code, http.StatusBadRequest)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "invalid_json")
}

func TestPathUUIDParsesValidSegment(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x/{userID}", nil)
	req.SetPathValue("userID", testUserID.String())
	id, err := pathUUID(req, "userID")
	requireNoErr(t, err)
	requireEqual(t, id, testUserID)
}

func TestPathUUIDRejectsInvalidSegment(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x/{userID}", nil)
	req.SetPathValue("userID", "not-a-uuid")
	_, err := pathUUID(req, "userID")
	if err == nil {
		t.Fatal("pathUUID accepted invalid segment")
	}
	rec := httptest.NewRecorder()
	writeServiceError(rec, req, err)
	requireEqual(t, rec.Code, http.StatusBadRequest)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "invalid_id")
	requireEqual(t, env.Err.Message, "invalid userID in path")
}

func TestPathUUIDRejectsMissingSegment(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if _, err := pathUUID(req, "userID"); err == nil {
		t.Fatal("pathUUID accepted missing segment")
	}
}

func TestParseOptionalUUIDAcceptsNilAndEmpty(t *testing.T) {
	id, err := parseOptionalUUID(nil)
	requireNoErr(t, err)
	if id != nil {
		t.Fatalf("nil input produced %v, want nil", id)
	}
	empty := ""
	id, err = parseOptionalUUID(&empty)
	requireNoErr(t, err)
	if id != nil {
		t.Fatalf("empty input produced %v, want nil", id)
	}
}

func TestParseOptionalUUIDParsesValid(t *testing.T) {
	raw := testAcctID.String()
	id, err := parseOptionalUUID(&raw)
	requireNoErr(t, err)
	if id == nil || *id != testAcctID {
		t.Fatalf("got %v, want %v", id, testAcctID)
	}
}

func TestParseOptionalUUIDRejectsInvalid(t *testing.T) {
	raw := "nope"
	_, err := parseOptionalUUID(&raw)
	if err == nil {
		t.Fatal("parseOptionalUUID accepted invalid id")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	writeServiceError(rec, req, err)
	requireEqual(t, rec.Code, http.StatusBadRequest)
	var env errorEnvelope
	decodeBody(t, rec.Result(), &env)
	requireEqual(t, env.Err.Code, "invalid_id")
	requireEqual(t, env.Err.Message, "invalid id")
}
