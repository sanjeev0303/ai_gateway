package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/application/guardrails"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

type memoryGuardrailsRepo struct {
	items  map[string]domain.Guardrail
	events []domain.GuardrailEvent
}

func newMemoryGuardrailsRepo() *memoryGuardrailsRepo {
	return &memoryGuardrailsRepo{
		items: make(map[string]domain.Guardrail),
	}
}

func (m *memoryGuardrailsRepo) Create(_ context.Context, g domain.Guardrail) error {
	m.items[g.ID] = g
	return nil
}

func (m *memoryGuardrailsRepo) Get(_ context.Context, orgID, id string) (domain.Guardrail, error) {
	g, ok := m.items[id]
	if !ok || g.OrganizationID != orgID {
		return domain.Guardrail{}, repositories.ErrNotFound
	}
	return g, nil
}

func (m *memoryGuardrailsRepo) List(_ context.Context, orgID string) ([]domain.Guardrail, error) {
	var list []domain.Guardrail
	for _, g := range m.items {
		if g.OrganizationID == orgID {
			list = append(list, g)
		}
	}
	return list, nil
}

func (m *memoryGuardrailsRepo) ListForEvaluation(_ context.Context, orgID string, target domain.GuardrailTargetType, targetID string) ([]domain.Guardrail, error) {
	var list []domain.Guardrail
	for _, g := range m.items {
		if g.OrganizationID == orgID && g.TargetType == target {
			if targetID == "" || (g.TargetID != nil && *g.TargetID == targetID) {
				list = append(list, g)
			}
		}
	}
	return list, nil
}

func (m *memoryGuardrailsRepo) Update(_ context.Context, g domain.Guardrail) error {
	if _, ok := m.items[g.ID]; !ok {
		return repositories.ErrNotFound
	}
	m.items[g.ID] = g
	return nil
}

func (m *memoryGuardrailsRepo) Delete(_ context.Context, orgID, id string) error {
	g, ok := m.items[id]
	if !ok || g.OrganizationID != orgID {
		return repositories.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *memoryGuardrailsRepo) CreateEvent(_ context.Context, ev domain.GuardrailEvent) error {
	m.events = append(m.events, ev)
	return nil
}

func (m *memoryGuardrailsRepo) ListEvents(_ context.Context, orgID string) ([]domain.GuardrailEvent, error) {
	var list []domain.GuardrailEvent
	for _, e := range m.events {
		if e.OrganizationID == orgID {
			list = append(list, e)
		}
	}
	return list, nil
}

func newGuardrailsTestServer(t *testing.T, st *controlStubs, repo *memoryGuardrailsRepo) *httptest.Server {
	t.Helper()
	svc := identity.NewService(st.deps())
	gSvc := guardrails.NewService(repo)
	eval := guardrails.NewEvaluator()

	return buildServer(t, Deps{
		Identity:        svc,
		Guardrails:      gSvc,
		GuardrailsEval:  eval,
		DevLoginEnabled: true,
	})
}

func TestGuardrails_RequiresAuth(t *testing.T) {
	st := newControlStubs()
	repo := newMemoryGuardrailsRepo()
	srv := newGuardrailsTestServer(t, st, repo)
	defer srv.Close()

	for _, path := range []string{
		"/v1/guardrails",
		"/v1/guardrails/events",
		"/v1/guardrails/some-id",
	} {
		resp := doJSON(t, srv, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestGuardrails_LifecycleAndLiveTesting(t *testing.T) {
	st := newControlStubs()
	repo := newMemoryGuardrailsRepo()
	srv := newGuardrailsTestServer(t, st, repo)
	defer srv.Close()

	cfgBytes, _ := json.Marshal(domain.PIIConfiguration{
		Types: []string{"email", "phone"},
	})

	// 1. Member cannot create guardrail (403)
	forbiddenResp := doJSON(t, srv, http.MethodPost, "/v1/guardrails", map[string]any{
		"name":          "Block PII",
		"description":   "Blocks email and phone",
		"target_type":   "global",
		"detector":      "pii",
		"action":        "block",
		"configuration": json.RawMessage(cfgBytes),
		"enabled":       true,
	}, bearerHeader(st, domain.RoleMember))
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("member create got %d, want 403", forbiddenResp.StatusCode)
	}
	forbiddenResp.Body.Close()

	adminHeader := bearerHeader(st, domain.RoleAdmin)
	createResp := doJSON(t, srv, http.MethodPost, "/v1/guardrails", map[string]any{
		"name":          "Block PII",
		"description":   "Blocks email and phone",
		"target_type":   "global",
		"detector":      "pii",
		"action":        "block",
		"configuration": json.RawMessage(cfgBytes),
		"enabled":       true,
	}, adminHeader)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("admin create got %d, want 201", createResp.StatusCode)
	}
	var created guardrailDTO
	decodeBody(t, createResp, &created)
	if created.ID == "" || created.Name != "Block PII" {
		t.Fatalf("unexpected created guardrail: %+v", created)
	}

	// 3. List guardrails
	listResp := doJSON(t, srv, http.MethodGet, "/v1/guardrails", nil, adminHeader)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list got %d, want 200", listResp.StatusCode)
	}
	var listBody struct {
		Items []guardrailDTO `json:"items"`
	}
	decodeBody(t, listResp, &listBody)
	if len(listBody.Items) != 1 {
		t.Fatalf("got %d guardrails, want 1", len(listBody.Items))
	}

	// 4. Test guardrail with live input (triggers on email)
	testResp := doJSON(t, srv, http.MethodPost, "/v1/guardrails/"+created.ID+"/test", map[string]any{
		"payload":     "Please send secrets to test@example.com immediately",
		"token_count": 12,
	}, adminHeader)
	if testResp.StatusCode != http.StatusOK {
		t.Fatalf("test endpoint got %d, want 200", testResp.StatusCode)
	}
	var testResult testGuardrailResponse
	decodeBody(t, testResp, &testResult)
	if !testResult.Triggered || testResult.Action != "block" || testResult.MatchedCategory != "email" {
		t.Fatalf("expected triggered block for email, got %+v", testResult)
	}

	// Clean input to test endpoint (passes)
	safeTestResp := doJSON(t, srv, http.MethodPost, "/v1/guardrails/"+created.ID+"/test", map[string]any{
		"payload":     "This is a safe query about cloud architecture",
		"token_count": 8,
	}, adminHeader)
	var safeResult testGuardrailResponse
	decodeBody(t, safeTestResp, &safeResult)
	if safeResult.Triggered {
		t.Fatalf("expected safe query to pass, triggered on %+v", safeResult)
	}

	// 5. Update guardrail
	patchResp := doJSON(t, srv, http.MethodPatch, "/v1/guardrails/"+created.ID, map[string]any{
		"name": "Updated PII Guardrail",
	}, adminHeader)
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch got %d, want 200", patchResp.StatusCode)
	}
	patchResp.Body.Close()

	// 6. Delete guardrail
	delResp := doJSON(t, srv, http.MethodDelete, "/v1/guardrails/"+created.ID, nil, adminHeader)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete got %d, want 204", delResp.StatusCode)
	}
	delResp.Body.Close()

	// Verify 404 after delete
	getResp := doJSON(t, srv, http.MethodGet, "/v1/guardrails/"+created.ID, nil, adminHeader)
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d after delete, want 404", getResp.StatusCode)
	}
	getResp.Body.Close()
}

func TestGuardrails_EventsList(t *testing.T) {
	st := newControlStubs()
	repo := newMemoryGuardrailsRepo()
	repo.events = []domain.GuardrailEvent{
		{
			ID:              uuid.New().String(),
			OrganizationID:  testOrgID.String(),
			GuardrailID:     "g-1",
			RequestID:       "req-1",
			Direction:       domain.GuardrailEventDirectionInput,
			Detector:        domain.GuardrailDetectorPII,
			Action:          domain.GuardrailActionBlock,
			MatchedCategory: "email",
			Timestamp:       time.Now().UTC(),
		},
	}
	srv := newGuardrailsTestServer(t, st, repo)
	defer srv.Close()

	adminHeader := bearerHeader(st, domain.RoleAdmin)
	resp := doJSON(t, srv, http.MethodGet, "/v1/guardrails/events", nil, adminHeader)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []guardrailEventDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 1 {
		t.Fatalf("got %d events, want 1", len(body.Items))
	}
	if body.Items[0].MatchedCategory != "email" || body.Items[0].Action != "block" {
		t.Fatalf("unexpected event: %+v", body.Items[0])
	}
}
