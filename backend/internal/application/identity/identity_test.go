package identity

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// Identity flows: user provisioning on first sight, organization
// provisioning with slug collision retry, and the dev login session issued
// on top of both.

func TestGetOrCreateUser(t *testing.T) {
	ctx := context.Background()
	user := domain.User{ID: testUserID, Email: "alice@example.com", DisplayName: "Alice"}

	t.Run("returns an existing user", func(t *testing.T) {
		svc, st := newTestService()
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return user, nil }
		got, err := svc.getOrCreateUser(ctx, user.Email, "ignored")
		requireNoErr(t, err)
		requireEqual(t, got.ID, user.ID)
	})

	t.Run("creates on first sight", func(t *testing.T) {
		svc, st := newTestService()
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) {
			return domain.User{}, repositories.ErrNotFound
		}
		st.users.create = func(_ context.Context, in domain.NewUser) (domain.User, error) {
			requireEqual(t, in.Email, user.Email)
			requireEqual(t, in.DisplayName, "Alice")
			return user, nil
		}
		got, err := svc.getOrCreateUser(ctx, user.Email, "Alice")
		requireNoErr(t, err)
		requireEqual(t, got.ID, user.ID)
	})

	t.Run("adopts the winner after a lost create race", func(t *testing.T) {
		svc, st := newTestService()
		created := false
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) {
			if created {
				return user, nil
			}
			return domain.User{}, repositories.ErrNotFound
		}
		st.users.create = func(_ context.Context, _ domain.NewUser) (domain.User, error) {
			created = true
			return domain.User{}, repositories.ErrConflict
		}
		got, err := svc.getOrCreateUser(ctx, user.Email, "")
		requireNoErr(t, err)
		requireEqual(t, got.ID, user.ID)
	})

	t.Run("propagates non-not-found lookup errors", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) {
			return domain.User{}, boom
		}
		_, err := svc.getOrCreateUser(ctx, user.Email, "")
		requireErrIs(t, err, boom)
	})

	t.Run("propagates non-conflict create errors", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) {
			return domain.User{}, repositories.ErrNotFound
		}
		st.users.create = func(_ context.Context, _ domain.NewUser) (domain.User, error) {
			return domain.User{}, boom
		}
		_, err := svc.getOrCreateUser(ctx, user.Email, "")
		requireErrIs(t, err, boom)
	})
}

func TestProvisionOrganization(t *testing.T) {
	ctx := context.Background()
	user := domain.User{ID: testUserID, Email: "alice@example.com", DisplayName: "Alice"}

	t.Run("slugs the display name and installs the owner", func(t *testing.T) {
		svc, st := newTestService()
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			requireEqual(t, in.Name, "My Team")
			requireEqual(t, in.Slug, "my-team")
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		added := false
		st.members.add = func(_ context.Context, orgID, memberID uuid.UUID, role string) (domain.OrganizationMember, error) {
			requireEqual(t, orgID, testOrgID)
			requireEqual(t, memberID, testUserID)
			requireEqual(t, role, domain.RoleOwner)
			added = true
			return domain.OrganizationMember{}, nil
		}
		org, err := svc.provisionOrganization(ctx, user, "My Team")
		requireNoErr(t, err)
		requireEqual(t, org.ID, testOrgID)
		if !added {
			t.Error("owner membership was not installed")
		}
	})

	t.Run("falls back to the email local part for blank names", func(t *testing.T) {
		svc, st := newTestService()
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			requireEqual(t, in.Name, "alice")
			requireEqual(t, in.Slug, "alice")
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, nil
		}
		_, err := svc.provisionOrganization(ctx, user, "  ")
		requireNoErr(t, err)
	})

	t.Run("falls back to the literal slug org", func(t *testing.T) {
		svc, st := newTestService()
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			requireEqual(t, in.Slug, "org")
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, nil
		}
		_, err := svc.provisionOrganization(ctx, user, "!!!")
		requireNoErr(t, err)
	})

	t.Run("retries a slug collision with a hex suffix", func(t *testing.T) {
		svc, st := newTestService()
		var created []domain.NewOrganization
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			created = append(created, in)
			if len(created) == 1 {
				return domain.Organization{}, repositories.ErrConflict
			}
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, nil
		}
		_, err := svc.provisionOrganization(ctx, user, "My Team")
		requireNoErr(t, err)
		if len(created) != 2 {
			t.Fatalf("create attempts = %d, want 2", len(created))
		}
		requireEqual(t, created[0].Slug, "my-team")
		suffix := strings.TrimPrefix(created[1].Slug, "my-team-")
		if len(suffix) != 4 {
			t.Errorf("collision suffix %q length = %d, want 4", suffix, len(suffix))
		}
		if _, err := hex.DecodeString(suffix); err != nil {
			t.Errorf("collision suffix %q is not hex: %v", suffix, err)
		}
	})

	t.Run("fails after exhausting slug candidates", func(t *testing.T) {
		svc, st := newTestService()
		st.orgs.create = func(_ context.Context, _ domain.NewOrganization) (domain.Organization, error) {
			return domain.Organization{}, repositories.ErrConflict
		}
		_, err := svc.provisionOrganization(ctx, user, "My Team")
		requireErrIs(t, err, repositories.ErrConflict)
	})

	t.Run("propagates non-conflict create errors without retrying", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		calls := 0
		st.orgs.create = func(_ context.Context, _ domain.NewOrganization) (domain.Organization, error) {
			calls++
			return domain.Organization{}, boom
		}
		_, err := svc.provisionOrganization(ctx, user, "My Team")
		requireErrIs(t, err, boom)
		if calls != 1 {
			t.Errorf("create calls = %d, want 1", calls)
		}
	})

	t.Run("propagates member installation errors", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, boom
		}
		_, err := svc.provisionOrganization(ctx, user, "My Team")
		requireErrIs(t, err, boom)
	})
}

func TestDevLogin(t *testing.T) {
	ctx := context.Background()
	user := domain.User{ID: testUserID, Email: "alice@example.com", DisplayName: "Alice"}

	t.Run("rejects a blank email", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.DevLogin(ctx, "   ", "Alice")
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("returns the first organization and its role", func(t *testing.T) {
		svc, st := newTestService()
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return user, nil }
		st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
			return []domain.UserOrganization{{
				Organization: domain.Organization{ID: testOrgID, Name: "Acme", Slug: "acme"},
				Role:         domain.RoleAdmin,
			}}, nil
		}
		st.issuer.issue = func(_ context.Context, claims domain.TokenClaims) (string, error) {
			requireEqual(t, claims.OrganizationID, testOrgID)
			requireEqual(t, claims.Role, domain.RoleAdmin)
			requireEqual(t, claims.Kind, domain.TokenKindUser)
			return "token-1", nil
		}
		sess, err := svc.DevLogin(ctx, " ALICE@Example.COM ", "Alice")
		requireNoErr(t, err)
		requireEqual(t, sess.Token, "token-1")
		requireEqual(t, sess.Claims.Subject, testUserID.String())
		requireEqual(t, sess.Claims.Email, "alice@example.com")
		requireEqual(t, sess.Claims.OrganizationID, testOrgID)
	})

	t.Run("provisions an organization and signs in as owner", func(t *testing.T) {
		svc, st := newTestService()
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return user, nil }
		st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
			return nil, nil
		}
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			requireEqual(t, in.Name, "alice")
			requireEqual(t, in.Slug, "alice")
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, nil
		}
		st.issuer.issue = func(_ context.Context, claims domain.TokenClaims) (string, error) {
			requireEqual(t, claims.OrganizationID, testOrgID)
			requireEqual(t, claims.Role, domain.RoleOwner)
			return "token-2", nil
		}
		sess, err := svc.DevLogin(ctx, "alice@example.com", "")
		requireNoErr(t, err)
		requireEqual(t, sess.Claims.Role, domain.RoleOwner)
		requireEqual(t, sess.Claims.Kind, domain.TokenKindUser)
	})

	t.Run("propagates organization listing failures", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return user, nil }
		st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
			return nil, boom
		}
		_, err := svc.DevLogin(ctx, "alice@example.com", "Alice")
		requireErrIs(t, err, boom)
	})

	t.Run("propagates issuer failures", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return user, nil }
		st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
			return nil, nil
		}
		st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
			return domain.Organization{ID: testOrgID, Name: in.Name, Slug: in.Slug}, nil
		}
		st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
			return domain.OrganizationMember{}, nil
		}
		st.issuer.issue = func(_ context.Context, _ domain.TokenClaims) (string, error) { return "", boom }
		_, err := svc.DevLogin(ctx, "alice@example.com", "Alice")
		requireErrIs(t, err, boom)
	})
}
