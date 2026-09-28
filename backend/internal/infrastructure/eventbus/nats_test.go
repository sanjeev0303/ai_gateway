package eventbus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
)

func TestUsageEventMessageJSONRoundTrip(t *testing.T) {
	orgID := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	credentialID := uuid.New()
	providerReqID := "prov-1"
	errorCode := "upstream_error"
	requestTime := time.Date(2026, 5, 6, 7, 8, 9, 123456789, time.UTC)

	original := ports.UsageEventMessage{
		OrganizationID: orgID,
		Request: domain.NewLlmRequest{
			ProjectID:         &projectID,
			UserID:            &userID,
			APICredentialID:   &credentialID,
			RequestID:         "req-1",
			ProviderRequestID: &providerReqID,
			ProviderKind:      domain.ProviderKindOpenAI,
			Model:             "gpt-4o",
			Status:            domain.LlmRequestStatusFailed,
			ErrorCode:         &errorCode,
			PromptTokens:      11,
			CompletionTokens:  22,
			CachedInputTokens: 3,
			TotalTokens:       33,
			LatencyMs:         44,
			CostMicros:        555,
			CreatedAt:         requestTime,
		},
		Usage: domain.NewUsageEvent{
			RequestID: "req-1",
			EventType: "completion",
			Quantity:  7,
			Unit:      "token",
			Metadata:  []byte(`{"region":"eu"}`),
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ports.UsageEventMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.OrganizationID != orgID {
		t.Fatalf("OrganizationID = %s, want %s", decoded.OrganizationID, orgID)
	}
	if !decoded.Request.CreatedAt.Equal(requestTime) {
		t.Fatalf("CreatedAt = %s, want %s", decoded.Request.CreatedAt, requestTime)
	}
	if decoded.Request.UserID == nil || *decoded.Request.UserID != userID {
		t.Fatalf("UserID did not survive the wire: %v", decoded.Request.UserID)
	}
	if decoded.Request.ProjectID == nil || *decoded.Request.ProjectID != projectID {
		t.Fatalf("ProjectID did not survive the wire: %v", decoded.Request.ProjectID)
	}
	if decoded.Request.APICredentialID == nil || *decoded.Request.APICredentialID != credentialID {
		t.Fatalf("APICredentialID did not survive the wire: %v", decoded.Request.APICredentialID)
	}
	if decoded.Request.ProviderRequestID == nil || *decoded.Request.ProviderRequestID != providerReqID {
		t.Fatalf("ProviderRequestID did not survive the wire: %v", decoded.Request.ProviderRequestID)
	}
	if decoded.Request.ErrorCode == nil || *decoded.Request.ErrorCode != errorCode {
		t.Fatalf("ErrorCode did not survive the wire: %v", decoded.Request.ErrorCode)
	}
	if decoded.Request.PromptTokens != 11 || decoded.Request.CompletionTokens != 22 ||
		decoded.Request.CachedInputTokens != 3 || decoded.Request.TotalTokens != 33 ||
		decoded.Request.LatencyMs != 44 || decoded.Request.CostMicros != 555 {
		t.Fatalf("counters did not survive the wire: %+v", decoded.Request)
	}
	if decoded.Usage.Quantity != 7 || decoded.Usage.Unit != "token" {
		t.Fatalf("usage did not survive the wire: %+v", decoded.Usage)
	}
	if string(decoded.Usage.Metadata) != `{"region":"eu"}` {
		t.Fatalf("Metadata did not survive the wire: %s", decoded.Usage.Metadata)
	}
}

func TestUsageEventMessageJSONRoundTripPreservesNilPointers(t *testing.T) {
	original := ports.UsageEventMessage{
		OrganizationID: uuid.New(),
		Request: domain.NewLlmRequest{
			RequestID:    "req-nil",
			ProviderKind: domain.ProviderKindOpenAI,
			Model:        "gpt-4o",
			Status:       domain.LlmRequestStatusSucceeded,
			CreatedAt:    time.Now().UTC(),
		},
		Usage: domain.NewUsageEvent{RequestID: "req-nil", EventType: "completion", Quantity: 1, Unit: "token"},
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ports.UsageEventMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Request.UserID != nil || decoded.Request.ProjectID != nil || decoded.Request.ErrorCode != nil {
		t.Fatalf("absent attribution must stay nil: %+v", decoded.Request)
	}
	if decoded.Usage.Metadata != nil {
		t.Fatalf("absent metadata must stay nil, got %s", decoded.Usage.Metadata)
	}
}

func TestRetryBackoffIsExponentialAndCapped(t *testing.T) {
	delays := retryBackoff()
	if len(delays) != natsMaxDeliver {
		t.Fatalf("len(delays) = %d, want %d", len(delays), natsMaxDeliver)
	}
	if delays[0] != natsRetryBase {
		t.Fatalf("delays[0] = %s, want %s", delays[0], natsRetryBase)
	}
	for i := 1; i < len(delays); i++ {
		if delays[i] < delays[i-1] {
			t.Fatalf("backoff must not decrease at %d: %s then %s", i, delays[i-1], delays[i])
		}
		if delays[i] > natsMaxRetryDelay {
			t.Fatalf("delays[%d] = %s exceeds cap %s", i, delays[i], natsMaxRetryDelay)
		}
	}
	if last := delays[len(delays)-1]; last != natsMaxRetryDelay {
		t.Fatalf("final delay = %s, want the cap %s", last, natsMaxRetryDelay)
	}
}

func TestNewNATSRejectsEmptyURL(t *testing.T) {
	if _, err := NewNATS(t.Context(), "", discardLogger()); err == nil {
		t.Fatal("an empty nats URL must be rejected")
	}
}
