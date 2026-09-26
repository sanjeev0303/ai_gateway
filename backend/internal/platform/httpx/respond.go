package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/platform/errors"
)

// ErrorBody is the JSON error envelope returned by every API endpoint
// (mirrors the ErrorEnvelope schema in backend/api/openapi/openapi.yaml).
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries one structured API error.
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

// WriteJSON encodes v with the shared JSON convention. The status code is
// written before encoding so headers are correct on success.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are already committed; encode failure can only be logged.
		slog.Default().Error("write json response", "error", err)
	}
}

// WriteError responds with the structured error envelope. requestID is
// echoed for correlation with logs; err is never logged here.
func WriteError(w http.ResponseWriter, status int, requestID string, err error) {
	detail := ErrorDetail{RequestID: requestID}
	if appErr, ok := err.(*errors.Error); ok {
		detail.Code = appErr.CodeOrKind()
		detail.Message = appErr.Message
	} else {
		detail.Code = errors.KindInternal.String()
		detail.Message = "internal error"
	}
	WriteJSON(w, status, ErrorBody{Error: detail})
}

// StatusFor maps an error kind to its canonical HTTP status code.
func StatusFor(kind errors.Kind) int {
	switch kind {
	case errors.KindInvalidArgument:
		return http.StatusBadRequest
	case errors.KindUnauthenticated:
		return http.StatusUnauthorized
	case errors.KindPermissionDenied:
		return http.StatusForbidden
	case errors.KindNotFound:
		return http.StatusNotFound
	case errors.KindConflict:
		return http.StatusConflict
	case errors.KindRateLimited:
		return http.StatusTooManyRequests
	case errors.KindTimeout:
		return http.StatusGatewayTimeout
	case errors.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
