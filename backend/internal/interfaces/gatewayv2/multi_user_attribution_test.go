package gatewayv2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// stubMultiUserAuth manages 3 separate users with individual credentials.
type stubMultiUserAuth struct {
	creds map[string]GatewayCredential
}

func (s *stubMultiUserAuth) Authenticate(ctx context.Context, rawKey string) (GatewayCredential, error) {
	cred, ok := s.creds[rawKey]
	if !ok {
		return GatewayCredential{}, ErrUnauthorized
	}
	if cred.Status == "revoked" {
		return GatewayCredential{}, ErrCredentialRevoked
	}
	return cred, nil
}

func TestMultiUser_AttributionAndIsolation(t *testing.T) {
	t.Log("Provider mode: Deterministic In-Memory Mock Provider (Hermetic Local Execution)")
	orgID := uuid.New()
	user1ID := uuid.New()
	user2ID := uuid.New()
	user3ID := uuid.New()

	cred1ID := uuid.New()
	cred2ID := uuid.New()
	cred3ID := uuid.New()

	projCoreID := uuid.New()
	projResearchID := uuid.New()

	keyUser1 := "aigw_user1_secret_key_alpha"
	keyUser2 := "aigw_user2_secret_key_beta"
	keyUser3 := "aigw_user3_secret_key_gamma"

	credsMap := map[string]GatewayCredential{
		keyUser1: {
			ID:             cred1ID,
			OrganizationID: orgID,
			SubjectType:    "user",
			SubjectID:      user1ID,
			ProjectID:      &projCoreID,
			Scopes:         []string{"gateway"},
			Status:         "active",
		},
		keyUser2: {
			ID:             cred2ID,
			OrganizationID: orgID,
			SubjectType:    "user",
			SubjectID:      user2ID,
			ProjectID:      &projCoreID,
			Scopes:         []string{"gateway"},
			Status:         "active",
		},
		keyUser3: {
			ID:             cred3ID,
			OrganizationID: orgID,
			SubjectType:    "user",
			SubjectID:      user3ID,
			ProjectID:      &projResearchID,
			Scopes:         []string{"gateway"},
			Status:         "active",
		},
	}

	authStub := &stubMultiUserAuth{creds: credsMap}
	tenantStub := &stubGatewayTenantResolver{
		upstream: ResolvedUpstream{
			ProviderAccount: ProviderAccount{
				ID:             uuid.New(),
				OrganizationID: orgID,
				ProviderKind:   "gemini",
				Status:         "active",
			},
			UpstreamKey: "sk-dummy-test-key",
		},
	}
	telemetryStub := &stubGatewayTelemetry{}
	proxyStub := &stubGatewayStreamProxy{
		usage: &StreamUsage{
			StatusCode:   http.StatusOK,
			Provider:     "gemini",
			InputTokens:  25,
			OutputTokens: 35,
			Reported:     true,
			FirstTokenAt: time.Now().UTC(),
		},
	}

	g := &Gateway{
		config:           DefaultGatewayConfig(),
		logger:           discardLogger(),
		authenticator:    authStub,
		tenantResolver:   tenantStub,
		policyResolver:   &stubGatewayPolicyResolver{snapshot: PolicySnapshot{OrganizationID: orgID}},
		rateLimiter:      &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true, Limit: 1000, Remaining: 999}},
		budgetGuard:      &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true, Limit: 1000}},
		requestInspector: NewRequestInspector(),
		providerRouter:   NewProviderRouter("", ""),
		streamProxy:      proxyStub,
		telemetry:        telemetryStub,
	}

	// 1. Request by User 1 (Project Gemini Core)
	body1 := `{"model": "gemini-2.5-flash", "messages": [{"role": "user", "content": "Request from User 1"}]}`
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body1))
	req1.Header.Set("x-api-key", keyUser1)
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()

	g.ServeHTTP(rec1, req1)
	requireEqual(t, rec1.Code, http.StatusOK)
	requireEqual(t, len(telemetryStub.usage), 1)

	u1 := telemetryStub.usage[0]
	requireEqual(t, u1.OrganizationID, orgID)
	requireEqual(t, u1.SubjectID, user1ID)
	requireEqual(t, u1.CredentialID, cred1ID)
	requireEqual(t, *u1.ProjectID, projCoreID)
	requireEqual(t, u1.InputTokens, 25)
	requireEqual(t, u1.OutputTokens, 35)
	requireEqual(t, u1.ProviderMeta["tokens_reported"], true)

	// 2. Request by User 2 (Project Gemini Core)
	body2 := `{"model": "gemini-2.5-flash", "messages": [{"role": "user", "content": "Request from User 2"}]}`
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body2))
	req2.Header.Set("x-api-key", keyUser2)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()

	g.ServeHTTP(rec2, req2)
	requireEqual(t, rec2.Code, http.StatusOK)
	requireEqual(t, len(telemetryStub.usage), 2)

	u2 := telemetryStub.usage[1]
	requireEqual(t, u2.OrganizationID, orgID)
	requireEqual(t, u2.SubjectID, user2ID)
	requireEqual(t, u2.CredentialID, cred2ID)
	requireEqual(t, *u2.ProjectID, projCoreID)

	// 3. Request by User 3 (Project Gemini Research)
	body3 := `{"model": "gemini-2.5-flash", "messages": [{"role": "user", "content": "Request from User 3"}]}`
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body3))
	req3.Header.Set("x-api-key", keyUser3)
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()

	g.ServeHTTP(rec3, req3)
	requireEqual(t, rec3.Code, http.StatusOK)
	requireEqual(t, len(telemetryStub.usage), 3)

	u3 := telemetryStub.usage[2]
	requireEqual(t, u3.OrganizationID, orgID)
	requireEqual(t, u3.SubjectID, user3ID)
	requireEqual(t, u3.CredentialID, cred3ID)
	requireEqual(t, *u3.ProjectID, projResearchID)

	// 4. Verification: User 1 cannot spoof User 2 ID via Header
	reqSpoofUser := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body1))
	reqSpoofUser.Header.Set("x-api-key", keyUser1)
	reqSpoofUser.Header.Set("X-User-ID", user2ID.String())
	reqSpoofUser.Header.Set("Content-Type", "application/json")
	recSpoofUser := httptest.NewRecorder()

	g.ServeHTTP(recSpoofUser, reqSpoofUser)
	requireEqual(t, recSpoofUser.Code, http.StatusForbidden)
	requireTrue(t, strings.Contains(recSpoofUser.Body.String(), "permission_error"), "user spoofing must be blocked")

	// 5. Verification: User 1 cannot spoof another Org ID via Header
	reqSpoofOrg := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body1))
	reqSpoofOrg.Header.Set("x-api-key", keyUser1)
	reqSpoofOrg.Header.Set("X-Organization-ID", uuid.New().String())
	reqSpoofOrg.Header.Set("Content-Type", "application/json")
	recSpoofOrg := httptest.NewRecorder()

	g.ServeHTTP(recSpoofOrg, reqSpoofOrg)
	requireEqual(t, recSpoofOrg.Code, http.StatusForbidden)
	requireTrue(t, strings.Contains(recSpoofOrg.Body.String(), "tenant_isolation_violation"), "cross-tenant spoofing must be blocked")

	// 6. Verification: User 2 key revocation halts only User 2
	c2 := credsMap[keyUser2]
	c2.Status = "revoked"
	credsMap[keyUser2] = c2

	recRevoked := httptest.NewRecorder()
	g.ServeHTTP(recRevoked, req2)
	requireEqual(t, recRevoked.Code, http.StatusUnauthorized)

	// User 1 and User 3 remain active and permitted
	recUser1After := httptest.NewRecorder()
	reqUser1Again := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body1))
	reqUser1Again.Header.Set("x-api-key", keyUser1)
	reqUser1Again.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(recUser1After, reqUser1Again)
	requireEqual(t, recUser1After.Code, http.StatusOK)
}

func TestMultiUser_ProviderRequestFormats(t *testing.T) {
	t.Log("Provider mode: Deterministic In-Memory Mock Provider (Hermetic Local Execution)")
	orgID := uuid.New()
	userID := uuid.New()
	credID := uuid.New()
	key := "aigw_valid_format_test_key"

	authStub := &stubMultiUserAuth{
		creds: map[string]GatewayCredential{
			key: {
				ID:             credID,
				OrganizationID: orgID,
				SubjectType:    "user",
				SubjectID:      userID,
				Scopes:         []string{"gateway"},
				Status:         "active",
			},
		},
	}

	telemetryStub := &stubGatewayTelemetry{}
	proxyStub := &stubGatewayStreamProxy{
		usage: &StreamUsage{
			StatusCode:   http.StatusOK,
			Provider:     "anthropic",
			InputTokens:  10,
			OutputTokens: 20,
			Reported:     true,
		},
	}

	g := &Gateway{
		config:           DefaultGatewayConfig(),
		logger:           discardLogger(),
		authenticator:    authStub,
		tenantResolver:   &stubGatewayTenantResolver{upstream: upstreamFixture()},
		policyResolver:   &stubGatewayPolicyResolver{snapshot: PolicySnapshot{OrganizationID: orgID}},
		rateLimiter:      &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true, Limit: 1000, Remaining: 999}},
		budgetGuard:      &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true, Limit: 1000}},
		requestInspector: NewRequestInspector(),
		providerRouter:   NewProviderRouter("", ""),
		streamProxy:      proxyStub,
		telemetry:        telemetryStub,
	}

	// 1. Anthropic format
	anthropicBody := `{"model": "claude-sonnet-4-5", "max_tokens": 1024, "messages": [{"role": "user", "content": "hi"}]}`
	reqA := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(anthropicBody))
	reqA.Header.Set("x-api-key", key)
	reqA.Header.Set("Content-Type", "application/json")
	recA := httptest.NewRecorder()
	g.ServeHTTP(recA, reqA)
	requireEqual(t, recA.Code, http.StatusOK)

	// 2. OpenAI format
	openAIBody := `{"model": "gpt-4o", "messages": [{"role": "user", "content": "hi"}], "stream": false}`
	reqO := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(openAIBody))
	reqO.Header.Set("Authorization", "Bearer "+key)
	reqO.Header.Set("Content-Type", "application/json")
	recO := httptest.NewRecorder()
	g.ServeHTTP(recO, reqO)
	requireEqual(t, recO.Code, http.StatusOK)

	// 3. Gemini format (under /v1/chat/completions or model gemini-*)
	geminiBody := `{"model": "gemini-2.5-flash", "messages": [{"role": "user", "content": "hi"}]}`
	reqG := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(geminiBody))
	reqG.Header.Set("x-api-key", key)
	reqG.Header.Set("Content-Type", "application/json")
	recG := httptest.NewRecorder()
	g.ServeHTTP(recG, reqG)
	requireEqual(t, recG.Code, http.StatusOK)
}

func TestMultiUser_LiveGeminiSmokeTestOrBlocked(t *testing.T) {
	liveKey := os.Getenv("GEMINI_API_KEY")
	if liveKey == "" {
		liveKey = os.Getenv("GOOGLE_API_KEY")
	}

	if liveKey == "" {
		t.Log("STATUS: [BLOCKED] Live Gemini smoke test skipped because GEMINI_API_KEY is not configured in local environment.")
		t.Skip("BLOCKED: GEMINI_API_KEY not configured")
		return
	}

	// When GEMINI_API_KEY is available, execute a live smoke test against Google Gemini API
	reqURL := "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	payload := map[string]any{
		"model": "gemini-2.5-flash",
		"messages": []map[string]string{
			{"role": "user", "content": "Ping Sentinel Smoke Test"},
		},
		"max_tokens": 16,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create smoke test request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+liveKey)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("live gemini request transport error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("live gemini smoke test rejected with HTTP %d", resp.StatusCode)
	}
	t.Log("STATUS: [PASS] Live Gemini smoke test succeeded with HTTP 200 OK")
}
