package repositories_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
)

// newTestStore connects to the database named by AIGW_TEST_DATABASE_URL and
// skips when it is unset, so the unit suite stays runnable without Postgres.
func newTestStore(t *testing.T) *postgres.Store {
	t.Helper()
	url := os.Getenv("AIGW_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set AIGW_TEST_DATABASE_URL to run the Postgres integration tests")
	}
	store, err := postgres.NewStore(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

// TestAuthRepositoryRegisterRollsBackWhenRefreshTokenInsertFails drives the one
// failure the atomic registration exists to survive: the refresh-token insert
// is the last of four writes, so by the time it fails the user, organization,
// and membership rows already exist inside the transaction. If the rollback
// were missing, a signup would leave a real account that can never obtain a
// session, with no error visible to the user.
func TestAuthRepositoryRegisterRollsBackWhenRefreshTokenInsertFails(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	auth := repositories.NewAuthRepository(store)

	const collidingHash = "colliding-refresh-digest"

	// A pre-existing user gives the conflicting refresh row a valid owner,
	// because auth_refresh_tokens.user_id is a foreign key.
	decoyID := uuid.New()
	_, err := store.Pool().Exec(ctx,
		`INSERT INTO users (id, email, display_name, status) VALUES ($1, $2, $3, 'active')`,
		decoyID, "decoy-"+decoyID.String()+"@example.com", "Decoy")
	if err != nil {
		t.Fatalf("seed decoy user: %v", err)
	}
	t.Cleanup(func() {
		store.Pool().Exec(context.Background(), `DELETE FROM users WHERE id = $1`, decoyID)
	})
	_, err = store.Pool().Exec(ctx,
		`INSERT INTO auth_refresh_tokens (id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), decoyID, collidingHash, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("seed conflicting refresh token: %v", err)
	}

	email := "rollback-" + decoyID.String() + "@example.com"
	slug := "rollback-" + decoyID.String()
	orgID := uuid.New()

	_, _, err = auth.Register(ctx, domain.Registration{
		User:                  domain.NewUser{Email: email, DisplayName: "Rollback"},
		PasswordHash:          "$pbkdf2-sha256$600000$c2FsdA==$a2V5",
		RefreshTokenHash:      collidingHash,
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
		OrganizationID:        orgID,
		OrganizationName:      "Rollback Org",
		OrganizationSlug:      slug,
	})
	if err == nil {
		t.Fatal("Register must fail when the refresh digest is already taken")
	}

	var users, orgs, members int
	store.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE email = $1`, email).Scan(&users)
	store.Pool().QueryRow(ctx, `SELECT count(*) FROM organizations WHERE id = $1`, orgID).Scan(&orgs)
	store.Pool().QueryRow(ctx,
		`SELECT count(*) FROM organization_members WHERE organization_id = $1`, orgID).Scan(&members)

	if users != 0 || orgs != 0 || members != 0 {
		t.Fatalf("partial registration survived: users=%d organizations=%d members=%d",
			users, orgs, members)
	}
}

// TestAuthRepositoryRegisterCommitsEveryRowTogether is the other half of the
// contract: a clean signup leaves the account, the tenant, the membership, and
// the refresh digest committed together, so the browser's session works
// immediately after the response.
func TestAuthRepositoryRegisterCommitsEveryRowTogether(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	auth := repositories.NewAuthRepository(store)

	orgID := uuid.New()
	email := "commit-" + orgID.String() + "@example.com"
	slug := "commit-" + orgID.String()
	hash := "digest-" + orgID.String()

	user, org, err := auth.Register(ctx, domain.Registration{
		User:                  domain.NewUser{Email: email, DisplayName: "Commit"},
		PasswordHash:          "$pbkdf2-sha256$600000$c2FsdA==$a2V5",
		RefreshTokenHash:      hash,
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
		OrganizationID:        orgID,
		OrganizationName:      "Commit Org",
		OrganizationSlug:      slug,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	t.Cleanup(func() {
		store.Pool().Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
		store.Pool().Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	if org.ID != orgID {
		t.Fatalf("returned organization %s, want the caller-supplied %s", org.ID, orgID)
	}

	var stored string
	err = store.Pool().QueryRow(ctx,
		`SELECT token_hash FROM auth_refresh_tokens WHERE user_id = $1`, user.ID).Scan(&stored)
	if err != nil {
		t.Fatalf("refresh digest was not committed with the account: %v", err)
	}
	if stored != hash {
		t.Fatalf("stored digest %q, want %q", stored, hash)
	}

	var role string
	err = store.Pool().QueryRow(ctx,
		`SELECT role FROM organization_members WHERE organization_id = $1 AND user_id = $2`,
		orgID, user.ID).Scan(&role)
	if err != nil {
		t.Fatalf("owner membership was not committed: %v", err)
	}
	if role != string(domain.RoleOwner) {
		t.Fatalf("membership role %q, want %q", role, domain.RoleOwner)
	}
}
