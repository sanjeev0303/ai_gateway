package guardrails

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
	"time"
)

type stubGuardrailsRepo struct {
	items  map[string]domain.Guardrail
	events []domain.GuardrailEvent
}

func newStubGuardrailsRepo() *stubGuardrailsRepo {
	return &stubGuardrailsRepo{
		items: make(map[string]domain.Guardrail),
	}
}

func (s *stubGuardrailsRepo) Create(_ context.Context, g domain.Guardrail) error {
	s.items[g.ID] = g
	return nil
}

func (s *stubGuardrailsRepo) Get(_ context.Context, orgID, id string) (domain.Guardrail, error) {
	g, ok := s.items[id]
	if !ok || g.OrganizationID != orgID {
		return domain.Guardrail{}, repositories.ErrNotFound
	}
	return g, nil
}

func (s *stubGuardrailsRepo) List(_ context.Context, orgID string) ([]domain.Guardrail, error) {
	var list []domain.Guardrail
	for _, g := range s.items {
		if g.OrganizationID == orgID {
			list = append(list, g)
		}
	}
	return list, nil
}

func (s *stubGuardrailsRepo) ListForEvaluation(_ context.Context, orgID string, target domain.GuardrailTargetType, targetID string) ([]domain.Guardrail, error) {
	var list []domain.Guardrail
	for _, g := range s.items {
		if g.OrganizationID == orgID && g.TargetType == target {
			if targetID == "" || (g.TargetID != nil && *g.TargetID == targetID) {
				list = append(list, g)
			}
		}
	}
	return list, nil
}

func (s *stubGuardrailsRepo) Update(_ context.Context, g domain.Guardrail) error {
	existing, ok := s.items[g.ID]
	if !ok || existing.OrganizationID != g.OrganizationID {
		return repositories.ErrNotFound
	}
	s.items[g.ID] = g
	return nil
}

func (s *stubGuardrailsRepo) Delete(_ context.Context, orgID, id string) error {
	existing, ok := s.items[id]
	if !ok || existing.OrganizationID != orgID {
		return repositories.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *stubGuardrailsRepo) CreateEvent(_ context.Context, ev domain.GuardrailEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func (s *stubGuardrailsRepo) ListEvents(_ context.Context, orgID string) ([]domain.GuardrailEvent, error) {
	var list []domain.GuardrailEvent
	for _, e := range s.events {
		if e.OrganizationID == orgID {
			list = append(list, e)
		}
	}
	return list, nil
}

type stubAuditsRepo struct {
	entries []domain.AuditLog
}

func (s *stubAuditsRepo) Append(_ context.Context, orgID uuid.UUID, entry domain.NewAuditLog) (domain.AuditLog, error) {
	log := domain.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Action:         entry.Action,
		ResourceType:   entry.ResourceType,
		ResourceID:     entry.ResourceID,
	}
	s.entries = append(s.entries, log)
	return log, nil
}

func (s *stubAuditsRepo) List(_ context.Context, _ uuid.UUID, _ domain.AuditLogFilter, _, _ int32) ([]domain.AuditLog, error) {
	return s.entries, nil
}

func (s *stubAuditsRepo) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.AuditLog, error) {
	return domain.AuditLog{}, nil
}

func (s *stubAuditsRepo) PurgeExpired(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}

func TestService_CRUDAndAudits(t *testing.T) {
	repo := newStubGuardrailsRepo()
	audits := &stubAuditsRepo{}
	svc := NewService(repo, audits)

	orgID := uuid.New()
	adminActor := domain.Actor{
		OrganizationID: orgID,
		UserID:         uuid.New(),
		Role:           domain.RoleAdmin,
		Kind:           domain.ActorKindUser,
	}
	memberActor := domain.Actor{
		OrganizationID: orgID,
		UserID:         uuid.New(),
		Role:           domain.RoleMember,
		Kind:           domain.ActorKindUser,
	}

	cfg, _ := json.Marshal(map[string]any{"types": []string{"email"}})
	newG := domain.Guardrail{
		Name:          "PII Guardrail",
		Description:   "Blocks email leakage",
		TargetType:    domain.GuardrailTargetGlobal,
		Detector:      domain.GuardrailDetectorPII,
		Action:        domain.GuardrailActionBlock,
		Configuration: cfg,
		Enabled:       true,
	}

	// 1. Member cannot create guardrail (RBAC)
	_, err := svc.Create(context.Background(), memberActor, newG)
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for member create, got %v", err)
	}

	// 2. Admin can create guardrail
	created, err := svc.Create(context.Background(), adminActor, newG)
	if err != nil {
		t.Fatalf("admin create failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID on created guardrail")
	}

	// Verify audit log appended
	if len(audits.entries) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(audits.entries))
	}
	if audits.entries[0].Action != "guardrail.create" {
		t.Errorf("expected guardrail.create action, got %s", audits.entries[0].Action)
	}

	// 3. Member cannot read guardrail without permission
	_, err = svc.Get(context.Background(), memberActor, created.ID)
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for member read without permission, got %v", err)
	}

	// Auditor can read guardrail
	auditorActor := domain.Actor{
		OrganizationID: orgID,
		UserID:         uuid.New(),
		Role:           domain.RoleAuditor,
		Kind:           domain.ActorKindUser,
	}
	got, err := svc.Get(context.Background(), auditorActor, created.ID)
	if err != nil {
		t.Fatalf("auditor read failed: %v", err)
	}
	if got.Name != newG.Name {
		t.Errorf("expected name %s, got %s", newG.Name, got.Name)
	}

	// 4. Admin updates guardrail
	got.Description = "Updated description"
	updated, err := svc.Update(context.Background(), adminActor, got)
	if err != nil {
		t.Fatalf("admin update failed: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected updated description, got %s", updated.Description)
	}
	if len(audits.entries) != 2 || audits.entries[1].Action != "guardrail.update" {
		t.Fatalf("expected guardrail.update audit record, got %+v", audits.entries)
	}

	// 5. Admin deletes guardrail
	err = svc.Delete(context.Background(), adminActor, created.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if len(audits.entries) != 3 || audits.entries[2].Action != "guardrail.delete" {
		t.Fatalf("expected guardrail.delete audit record, got %+v", audits.entries)
	}

	// Confirm deleted
	_, err = svc.Get(context.Background(), adminActor, created.ID)
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}
