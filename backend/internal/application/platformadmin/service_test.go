package platformadmin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// The fakes embed the port interfaces so they satisfy them; any method a test
// does not expect to be called panics on the nil embedded value.

type fakeUsers struct {
	repositories.UserRepository
	users map[uuid.UUID]domain.User
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) ListOrganizationsForUser(context.Context, uuid.UUID) ([]domain.UserOrganization, error) {
	return nil, nil
}

type fakeRepo struct {
	repositories.PlatformAdminRepository
	orgStatus  map[uuid.UUID]string
	userStatus map[uuid.UUID]string
	audit      []domain.NewPlatformAuditEntry
}

func (f *fakeRepo) SetOrganizationStatus(_ context.Context, id uuid.UUID, status string) error {
	f.orgStatus[id] = status
	return nil
}

func (f *fakeRepo) SetUserStatus(_ context.Context, id uuid.UUID, status string) error {
	f.userStatus[id] = status
	return nil
}

func (f *fakeRepo) AppendAudit(_ context.Context, e domain.NewPlatformAuditEntry) error {
	f.audit = append(f.audit, e)
	return nil
}

func (f *fakeRepo) UserGatewayCredentials(context.Context, uuid.UUID) ([]domain.PlatformCredentialRef, error) {
	return nil, nil
}

type fakeKeys struct {
	repositories.GatewayCredentialRepository
	keys []domain.GatewayCredential
}

func (f *fakeKeys) List(context.Context, uuid.UUID) ([]domain.GatewayCredential, error) {
	return f.keys, nil
}

type fakeRefresh struct {
	repositories.RefreshTokenRepository
	revokedFor []uuid.UUID
}

func (f *fakeRefresh) RevokeAllForUser(_ context.Context, id uuid.UUID) error {
	f.revokedFor = append(f.revokedFor, id)
	return nil
}

type fakeInvalidator struct{ evicted []uuid.UUID }

func (f *fakeInvalidator) InvalidateCredential(_, credID uuid.UUID) {
	f.evicted = append(f.evicted, credID)
}

type env struct {
	svc     *Service
	repo    *fakeRepo
	inval   *fakeInvalidator
	refresh *fakeRefresh
	admin   domain.User
	member  domain.User
	keys    *fakeKeys
}

func newEnv() env {
	admin := domain.User{ID: uuid.New(), Email: "Root@Example.com", Status: domain.UserStatusActive}
	member := domain.User{ID: uuid.New(), Email: "member@example.com", Status: domain.UserStatusActive}
	repo := &fakeRepo{orgStatus: map[uuid.UUID]string{}, userStatus: map[uuid.UUID]string{}}
	inval := &fakeInvalidator{}
	refresh := &fakeRefresh{}
	keys := &fakeKeys{}
	svc := NewService(ServiceDeps{
		Repo:               repo,
		Users:              &fakeUsers{users: map[uuid.UUID]domain.User{admin.ID: admin, member.ID: member}},
		GatewayCredentials: keys,
		RefreshTokens:      refresh,
		Invalidator:        inval,
		AdminEmails:        []string{" root@example.com "},
	})
	return env{svc: svc, repo: repo, inval: inval, refresh: refresh, admin: admin, member: member, keys: keys}
}

func userActor(u domain.User) domain.Actor {
	return domain.Actor{Kind: domain.ActorKindUser, UserID: u.ID, OrganizationID: uuid.New()}
}

func TestAuthorize(t *testing.T) {
	e := newEnv()

	got, err := e.svc.Authorize(context.Background(), userActor(e.admin))
	if err != nil || got.ID != e.admin.ID {
		t.Fatalf("allowlisted admin (email case differs) was refused: %v", err)
	}

	if _, err := e.svc.Authorize(context.Background(), userActor(e.member)); !errors.Is(err, ErrNotPlatformAdmin) {
		t.Fatalf("non-allowlisted user got %v, want ErrNotPlatformAdmin", err)
	}

	service := domain.Actor{Kind: domain.ActorKindService, UserID: e.admin.ID, OrganizationID: uuid.New()}
	if _, err := e.svc.Authorize(context.Background(), service); !errors.Is(err, ErrNotPlatformAdmin) {
		t.Fatalf("service actor got %v, want ErrNotPlatformAdmin", err)
	}

	unknown := domain.Actor{Kind: domain.ActorKindUser, UserID: uuid.New()}
	if _, err := e.svc.Authorize(context.Background(), unknown); !errors.Is(err, ErrNotPlatformAdmin) {
		t.Fatalf("unknown user got %v, want ErrNotPlatformAdmin", err)
	}
}

func TestAuthorize_DisabledAdminRefused(t *testing.T) {
	e := newEnv()
	disabled := e.admin
	disabled.Status = domain.UserStatusDisabled
	e.svc.users = &fakeUsers{users: map[uuid.UUID]domain.User{disabled.ID: disabled}}

	if _, err := e.svc.Authorize(context.Background(), userActor(disabled)); !errors.Is(err, ErrNotPlatformAdmin) {
		t.Fatalf("disabled admin got %v, want ErrNotPlatformAdmin", err)
	}
}

func TestAuthorize_EmptyAllowlistDisablesConsole(t *testing.T) {
	e := newEnv()
	e.svc.admins = map[string]bool{}
	if _, err := e.svc.Authorize(context.Background(), userActor(e.admin)); !errors.Is(err, ErrNotPlatformAdmin) {
		t.Fatalf("got %v with an empty allowlist, want ErrNotPlatformAdmin", err)
	}
}

func TestSuspendOrganization_EvictsActiveKeysAndAudits(t *testing.T) {
	e := newEnv()
	orgID := uuid.New()
	active := domain.GatewayCredential{ID: uuid.New(), Status: domain.GatewayCredentialStatusActive}
	revoked := domain.GatewayCredential{ID: uuid.New(), Status: domain.GatewayCredentialStatusRevoked}
	e.keys.keys = []domain.GatewayCredential{active, revoked}

	if err := e.svc.SuspendOrganization(context.Background(), e.admin, orgID, "chargeback"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if e.repo.orgStatus[orgID] != domain.OrgStatusSuspended {
		t.Fatalf("org status = %q, want suspended", e.repo.orgStatus[orgID])
	}
	if len(e.inval.evicted) != 1 || e.inval.evicted[0] != active.ID {
		t.Fatalf("evicted %v, want only the active key %s", e.inval.evicted, active.ID)
	}
	if len(e.repo.audit) != 1 || e.repo.audit[0].Action != "platform.organization_suspended" || e.repo.audit[0].Reason != "chargeback" {
		t.Fatalf("unexpected audit: %+v", e.repo.audit)
	}
}

func TestActionsRequireReason(t *testing.T) {
	e := newEnv()
	if err := e.svc.SuspendOrganization(context.Background(), e.admin, uuid.New(), "   "); err == nil {
		t.Fatal("suspend without a reason succeeded")
	}
	if len(e.repo.orgStatus) != 0 {
		t.Fatal("org status changed despite the missing reason")
	}
}

func TestDisableUser(t *testing.T) {
	e := newEnv()

	if err := e.svc.DisableUser(context.Background(), e.admin, e.admin.ID, "testing"); err == nil {
		t.Fatal("an admin could disable their own account")
	}

	if err := e.svc.DisableUser(context.Background(), e.admin, e.member.ID, "left the company"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if e.repo.userStatus[e.member.ID] != domain.UserStatusDisabled {
		t.Fatalf("user status = %q, want disabled", e.repo.userStatus[e.member.ID])
	}
	if len(e.refresh.revokedFor) != 1 || e.refresh.revokedFor[0] != e.member.ID {
		t.Fatalf("refresh tokens revoked for %v, want the member", e.refresh.revokedFor)
	}
}
