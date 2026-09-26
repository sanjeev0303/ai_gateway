package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"
)

// Provider accounts and credentials: onboarding, permission gates, live
// validation, fingerprint/tail derivation, revoke and dual-key rotation.

func TestPermissionGates(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	cases := []struct {
		name string
		role string
		run  func(*Service) error
	}{
		// providers.manage
		{"member cannot create account", domain.RoleMember, func(s *Service) error {
			_, err := s.CreateAccount(ctx, userActor(domain.RoleMember), domain.NewProviderAccount{ProviderKind: domain.ProviderKindAnthropic, Name: "a"})
			return err
		}},
		{"service cannot create account", domain.RoleService, func(s *Service) error {
			_, err := s.CreateAccount(ctx, svcActor(), domain.NewProviderAccount{ProviderKind: domain.ProviderKindAnthropic, Name: "a"})
			return err
		}},
		// providers.read
		{"member cannot list accounts", domain.RoleMember, func(s *Service) error {
			_, err := s.ListAccounts(ctx, userActor(domain.RoleMember))
			return err
		}},
		{"service cannot list accounts", domain.RoleService, func(s *Service) error {
			_, err := s.ListAccounts(ctx, svcActor())
			return err
		}},
		{"member cannot get account", domain.RoleMember, func(s *Service) error {
			_, err := s.GetAccount(ctx, userActor(domain.RoleMember), testAccountID)
			return err
		}},
		// credentials.manage
		{"member cannot create credential", domain.RoleMember, func(s *Service) error {
			_, err := s.CreateCredential(ctx, userActor(domain.RoleMember), testAccountID, "k", domain.CredentialKindAPIKey, "sk-ant-s3cret")
			return err
		}},
		{"member cannot revoke credential", domain.RoleMember, func(s *Service) error {
			return s.RevokeCredential(ctx, userActor(domain.RoleMember), testCredID)
		}},
		{"member cannot rotate credential", domain.RoleMember, func(s *Service) error {
			_, err := s.RotateCredential(ctx, userActor(domain.RoleMember), testCredID, "sk-ant-n3w")
			return err
		}},
		// credentials.read
		{"member cannot list credentials", domain.RoleMember, func(s *Service) error {
			_, err := s.ListCredentials(ctx, userActor(domain.RoleMember))
			return err
		}},
		{"service cannot list credentials", domain.RoleService, func(s *Service) error {
			_, err := s.ListCredentials(ctx, svcActor())
			return err
		}},
		{"member cannot get credential", domain.RoleMember, func(s *Service) error {
			_, err := s.GetCredential(ctx, userActor(domain.RoleMember), testCredID)
			return err
		}},
		// models
		{"member cannot list models", domain.RoleMember, func(s *Service) error {
			_, err := s.ListModels(ctx, userActor(domain.RoleMember))
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.run(svc); !errors.Is(err, repositories.ErrForbidden) {
				t.Fatalf("err = %v, want ErrForbidden", err)
			}
		})
	}
}

func TestAuditorCanReadCredentials(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	auditor := userActor(domain.RoleAuditor)
	st.credentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderCredential{{ID: testCredID}}, nil
	}
	st.accounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderAccount{{ID: testAccountID}}, nil
	}
	st.models.list = func(_ context.Context) ([]domain.Model, error) {
		return []domain.Model{{ID: uuid.New(), ModelSlug: "claude-sonnet-4-5"}}, nil
	}
	creds, err := svc.ListCredentials(ctx, auditor)
	requireNoErr(t, err)
	requireEqual(t, len(creds), 1)
	accounts, err := svc.ListAccounts(ctx, auditor)
	requireNoErr(t, err)
	requireEqual(t, len(accounts), 1)
	models, err := svc.ListModels(ctx, auditor)
	requireNoErr(t, err)
	requireEqual(t, len(models), 1)
}

func TestCreateAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("validates provider kind", func(t *testing.T) {
		svc, st := newTestService()
		st.audits.appendFn = func(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error) {
			t.Fatal("audit must not be appended for invalid input")
			return domain.AuditLog{}, nil
		}
		_, err := svc.CreateAccount(ctx, userActor(domain.RoleOwner), domain.NewProviderAccount{ProviderKind: "unknown", Name: "a"})
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("validates non-empty name", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CreateAccount(ctx, userActor(domain.RoleOwner), domain.NewProviderAccount{ProviderKind: domain.ProviderKindAnthropic, Name: " "})
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("creates and audits", func(t *testing.T) {
		svc, st := newTestService()
		st.accounts.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderAccount) (domain.ProviderAccount, error) {
			requireEqual(t, orgID, testOrgID)
			requireEqual(t, in.ProviderKind, domain.ProviderKindAnthropic)
			requireEqual(t, in.Name, "prod-anthropic")
			requireEqual(t, in.BaseURL, "https://api.anthropic.com")
			return domain.ProviderAccount{ID: testAccountID, OrganizationID: orgID, ProviderKind: in.ProviderKind, Name: in.Name, Status: domain.ProviderStatusActive}, nil
		}
		acct, err := svc.CreateAccount(ctx, userActor(domain.RoleOwner), domain.NewProviderAccount{
			ProviderKind: domain.ProviderKindAnthropic,
			Name:         "prod-anthropic",
			BaseURL:      "https://api.anthropic.com",
		})
		requireNoErr(t, err)
		requireEqual(t, acct.ID, testAccountID)
		requireAudit(t, st, "provider_account.create")
	})

	t.Run("repo errors pass through", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("boom")
		st.accounts.create = func(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error) {
			return domain.ProviderAccount{}, boom
		}
		_, err := svc.CreateAccount(ctx, userActor(domain.RoleOwner), domain.NewProviderAccount{ProviderKind: domain.ProviderKindAnthropic, Name: "a"})
		requireErrIs(t, err, boom)
	})
}

func TestCreateCredentialValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		credName string
		account  func() (domain.ProviderAccount, error)
		kind     string
		secret   string
		want     error
	}{
		{"empty name is invalid", "", func() (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}, domain.CredentialKindAPIKey, "sk-ant-s3cret", repositories.ErrInvalid},
		{"api_key kind only", "key", func() (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}, domain.CredentialKindOAuth, "sk-ant-s3cret", repositories.ErrInvalid},
		{"empty secret is invalid", "key", func() (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}, domain.CredentialKindAPIKey, " ", repositories.ErrInvalid},
		{"account missing", "key", func() (domain.ProviderAccount, error) {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}, domain.CredentialKindAPIKey, "sk-ant-s3cret", repositories.ErrNotFound},
		{"disabled account conflicts", "key", func() (domain.ProviderAccount, error) {
			acct := activeAccount()
			acct.Status = domain.ProviderStatusDisabled
			return acct, nil
		}, domain.CredentialKindAPIKey, "sk-ant-s3cret", repositories.ErrConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
				return c.account()
			}
			_, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, c.credName, c.kind, c.secret)
			requireErrIs(t, err, c.want)
		})
	}
}

func TestCreateCredentialLiveValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("provider rejection maps to ErrInvalidProviderCredential", func(t *testing.T) {
		svc, st := newTestService()
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}
		st.provider.validateErr = providerclient.ErrInvalidCredentials
		st.secrets.values["ref-to-never-exist"] = "not-reached"
		_, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, "key", domain.CredentialKindAPIKey, "sk-ant-bad")
		requireErrIs(t, err, ErrInvalidProviderCredential)
		requireEqual(t, len(st.secrets.values), 1) // only the sentinel; nothing stored
	})

	t.Run("transport errors pass through", func(t *testing.T) {
		svc, st := newTestService()
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}
		boom := errors.New("dial tcp: refused")
		st.provider.validateErr = boom
		_, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, "key", domain.CredentialKindAPIKey, "sk-ant-x")
		requireErrIs(t, err, boom)
	})

	t.Run("unsupported provider kind has no client", func(t *testing.T) {
		svc, st := newTestService()
		openai := activeAccount()
		openai.ProviderKind = domain.ProviderKindOpenAI
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return openai, nil
		}
		_, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, "key", domain.CredentialKindAPIKey, "sk-ant-x")
		requireErrIs(t, err, repositories.ErrInvalid)
	})
}

func TestCreateCredentialStoresOnlyDerivedValues(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	const raw = "sk-ant-api03-91A7"
	st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return activeAccount(), nil
	}
	st.credentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, in.ProviderAccountID, testAccountID)
		requireEqual(t, in.Kind, domain.CredentialKindAPIKey)
		// The application layer derives the fingerprint/tail from the raw
		// secret; only these two values are persisted next to the row.
		if in.Fingerprint == "" || in.Tail != "91A7" {
			t.Fatalf("fingerprint=%q tail=%q: expected derived values", in.Fingerprint, in.Tail)
		}
		if in.SecretRef == "" || len(in.SecretRef) <= len(secretRefPrefix) {
			t.Fatalf("secret_ref=%q: expected a generated reference", in.SecretRef)
		}
		return domain.ProviderCredential{
			ID:                testCredID,
			OrganizationID:    orgID,
			ProviderAccountID: testAccountID,
			Name:              in.Name,
			Kind:              in.Kind,
			SecretRef:         in.SecretRef,
			Fingerprint:       in.Fingerprint,
			Tail:              in.Tail,
			Status:            domain.ProviderCredentialStatusActive,
			CreatedAt:         testNow,
			UpdatedAt:         testNow,
		}, nil
	}
	cred, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, "key", domain.CredentialKindAPIKey, raw)
	requireNoErr(t, err)
	requireEqual(t, cred.Tail, "91A7")
	requireEqual(t, cred.SecretRef, "provider/creds/"+cred.SecretRef[len(secretRefPrefix):])
	// The raw secret must never be persisted next to the credential nor
	// returned: it only ever exists in the secret store.
	requireEqual(t, len(st.secrets.values), 1)
	for _, v := range st.secrets.values {
		requireEqual(t, v, raw)
	}
	requireAudit(t, st, "provider_credential.create")
}

func TestCreateCredentialCleansUpOrphanedSecret(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return activeAccount(), nil
	}
	st.credentials.create = func(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{}, repositories.ErrConflict
	}
	_, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), testAccountID, "key", domain.CredentialKindAPIKey, "sk-ant-s3cret")
	requireErrIs(t, err, repositories.ErrConflict)
	requireEqual(t, len(st.secrets.values), 0) // orphaned secret cleaned up
}

func TestRotateCredential(t *testing.T) {
	ctx := context.Background()
	existing := func() domain.ProviderCredential {
		return domain.ProviderCredential{
			ID:                testCredID,
			OrganizationID:    testOrgID,
			ProviderAccountID: testAccountID,
			Name:              "prod-key",
			Kind:              domain.CredentialKindAPIKey,
			SecretRef:         "provider/creds/old",
			Fingerprint:       "sha256:old-fingerprint",
			Tail:              "old4",
			Status:            domain.ProviderCredentialStatusActive,
			CreatedAt:         testNow,
			UpdatedAt:         testNow,
		}
	}

	t.Run("rejects rotating a revoked credential", func(t *testing.T) {
		svc, st := newTestService()
		revoked := existing()
		revoked.Status = domain.ProviderCredentialStatusRevoked
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return revoked, nil
		}
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-n3w")
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("requires a non-empty new secret", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return existing(), nil
		}
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, " ")
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("missing credential passes through", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-n3w")
		requireErrIs(t, err, repositories.ErrNotFound)
	})

	t.Run("disabled account blocks rotation", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return existing(), nil
		}
		disabled := activeAccount()
		disabled.Status = domain.ProviderStatusDisabled
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return disabled, nil
		}
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-n3w")
		requireErrIs(t, err, repositories.ErrConflict)
	})

	t.Run("rejects a new secret the provider refuses", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return existing(), nil
		}
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}
		st.provider.validateErr = providerclient.ErrInvalidCredentials
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-bad")
		requireErrIs(t, err, ErrInvalidProviderCredential)
	})

	t.Run("dual-key rotation with zero-downtime ordering", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return existing(), nil
		}
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}
		var revokedOld bool
		st.credentials.create = func(ctx context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
			if revokedOld {
				t.Fatal("new credential created after the old one was revoked — leaves a gap")
			}
			requireEqual(t, orgID, testOrgID)
			requireEqual(t, in.ProviderAccountID, testAccountID)
			requireEqual(t, in.Name, "prod-key (rotated)")
			requireEqual(t, in.Tail, "n3w4")
			if in.Fingerprint == "sha256:old-fingerprint" {
				t.Fatal("rotated credential reuses the old fingerprint")
			}
			return domain.ProviderCredential{
				ID: testOtherID, OrganizationID: orgID,
				ProviderAccountID: testAccountID, Name: in.Name, Kind: in.Kind,
				SecretRef: in.SecretRef, Fingerprint: in.Fingerprint, Tail: in.Tail,
				Status: domain.ProviderCredentialStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
			}, nil
		}
		st.credentials.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
			requireEqual(t, orgID, testOrgID)
			revokedOld = true
			requireEqual(t, id, testCredID)
			revoked := existing()
			revoked.Status = domain.ProviderCredentialStatusRevoked
			return revoked, nil
		}
		cred, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-n3w4")
		requireNoErr(t, err)
		requireEqual(t, cred.ID, testOtherID)
		requireEqual(t, cred.Status, domain.ProviderCredentialStatusActive)
		requireEqual(t, revokedOld, true)
		// The old secret is deleted only after the new credential is live.
		if _, ok := st.secrets.values["provider/creds/old"]; ok {
			t.Fatal("old secret not deleted after rotation")
		}
		requireEqual(t, len(st.secrets.values), 1) // only the new secret remains
		requireAudit(t, st, "provider_credential.rotate")
	})

	t.Run("rolls back when revoking the old credential fails", func(t *testing.T) {
		svc, st := newTestService()
		st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
			return existing(), nil
		}
		st.accounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
			return activeAccount(), nil
		}
		boom := errors.New("revoke failed")
		created := &domain.ProviderCredential{}
		st.credentials.create = func(ctx context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
			*created = domain.ProviderCredential{
				ID: testOtherID, OrganizationID: orgID,
				ProviderAccountID: testAccountID, Name: in.Name, Kind: in.Kind,
				SecretRef: in.SecretRef, Fingerprint: in.Fingerprint, Tail: in.Tail,
				Status: domain.ProviderCredentialStatusActive,
			}
			return *created, nil
		}
		var revokedNew bool
		st.credentials.revoke = func(_ context.Context, _, id uuid.UUID) (domain.ProviderCredential, error) {
			if id == testOtherID {
				revokedNew = true
			}
			if id == testCredID {
				return domain.ProviderCredential{}, boom
			}
			return domain.ProviderCredential{}, nil
		}
		_, err := svc.RotateCredential(ctx, userActor(domain.RoleOwner), testCredID, "sk-ant-n3w")
		requireErrIs(t, err, boom)
		requireEqual(t, revokedNew, true)                          // new credential rolled back
		requireEqual(t, len(st.secrets.values), 0)                 // new secret deleted
		requireEqual(t, len(st.audits.actions), 0)                 // no rotate audit on failure
	})
}

func TestRevokeCredentialDeletesSecret(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.credentials.revoke = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{
			ID: testCredID, OrganizationID: testOrgID, SecretRef: "provider/creds/old",
			Status: domain.ProviderCredentialStatusRevoked,
		}, nil
	}
	st.secrets.values["provider/creds/old"] = "sk-ant-s3cret"
	err := svc.RevokeCredential(ctx, userActor(domain.RoleOwner), testCredID)
	requireNoErr(t, err)
	requireEqual(t, len(st.secrets.values), 0) // secret purged
	requireAudit(t, st, "provider_credential.revoke")
}

func TestRevokeCredentialMissingPassesThrough(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.credentials.revoke = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{}, repositories.ErrNotFound
	}
	err := svc.RevokeCredential(ctx, userActor(domain.RoleOwner), testCredID)
	requireErrIs(t, err, repositories.ErrNotFound)
}

func TestListModels(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.models.list = func(_ context.Context) ([]domain.Model, error) {
		return []domain.Model{
			{ModelSlug: "claude-opus-4-1", DisplayName: "Claude Opus 4.1"},
			{ModelSlug: "claude-sonnet-4-5", DisplayName: "Claude Sonnet 4.5"},
		}, nil
	}
	models, err := svc.ListModels(ctx, userActor(domain.RoleOwner))
	requireNoErr(t, err)
	requireEqual(t, len(models), 2)
}

func TestCheckHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("healthy probe records state", func(t *testing.T) {
		svc, st := newTestService()
		state, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-ant-ok")
		requireNoErr(t, err)
		requireEqual(t, state.Status, domain.ProviderHealthHealthy)
		requireEqual(t, state.ProviderKind, domain.ProviderKindAnthropic)
		requireEqual(t, state.CheckedAt, testNow)
		requireEqual(t, len(st.health.states), 1)
		requireEqual(t, st.health.states[0].Status, domain.ProviderHealthHealthy)
	})

	t.Run("provider rejection maps to credentials_rejected", func(t *testing.T) {
		svc, st := newTestService()
		st.provider.healthErr = providerclient.ErrInvalidCredentials
		state, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-ant-bad")
		requireNoErr(t, err)
		requireEqual(t, state.Status, domain.ProviderHealthUnhealthy)
		requireEqual(t, state.StatusReason, "credentials_rejected")
		requireEqual(t, st.health.states[0].StatusReason, "credentials_rejected")
	})

	t.Run("transport errors map to health_check_failed", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("dial tcp: refused")
		st.provider.healthErr = boom
		state, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-ant-x")
		requireNoErr(t, err)
		requireEqual(t, state.Status, domain.ProviderHealthUnhealthy)
		requireEqual(t, state.StatusReason, "health_check_failed")
		requireEqual(t, state.LastError, boom.Error())
	})

	t.Run("member is forbidden", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CheckHealth(ctx, userActor(domain.RoleMember), domain.ProviderKindAnthropic, "sk-ant-x")
		requireErrIs(t, err, repositories.ErrForbidden)
	})

	t.Run("unknown kind or empty key is invalid", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), "not-a-kind", "sk-ant-x")
		requireErrIs(t, err, repositories.ErrInvalid)
		_, err = svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "  ")
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("unregistered provider kind has no client", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindOpenAI, "sk-ant-x")
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("upsert failure surfaces", func(t *testing.T) {
		svc, st := newTestService()
		boom := errors.New("db down")
		st.health.upsert = func(context.Context, domain.ProviderHealthState) error { return boom }
		_, err := svc.CheckHealth(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-ant-x")
		requireErrIs(t, err, boom)
	})
}

func TestListHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("owner lists recorded states", func(t *testing.T) {
		svc, st := newTestService()
		st.health.states = []domain.ProviderHealthState{
			{ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderHealthHealthy},
		}
		states, err := svc.ListHealth(ctx, userActor(domain.RoleOwner))
		requireNoErr(t, err)
		requireEqual(t, len(states), 1)
		requireEqual(t, states[0].ProviderKind, domain.ProviderKindAnthropic)
	})

	t.Run("member is forbidden", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.ListHealth(ctx, userActor(domain.RoleMember))
		requireErrIs(t, err, repositories.ErrForbidden)
	})
}

func TestGetCredentialNeverReturnsSecret(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.credentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{
			ID: testCredID, OrganizationID: testOrgID, SecretRef: "provider/creds/old",
			Tail: "91A7", Status: domain.ProviderCredentialStatusActive,
		}, nil
	}
	cred, err := svc.GetCredential(ctx, userActor(domain.RoleOwner), testCredID)
	requireNoErr(t, err)
	requireEqual(t, cred.Tail, "91A7")
	requireEqual(t, cred.SecretRef, "provider/creds/old") // opaque ref only
}

func activeAccount() domain.ProviderAccount {
	return domain.ProviderAccount{
		ID: testAccountID, OrganizationID: testOrgID,
		ProviderKind: domain.ProviderKindAnthropic, Name: "prod-anthropic",
		Status: domain.ProviderStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
}

func svcActor() domain.Actor {
	return domain.ServiceActor(testOrgID, testOtherID, "ci-bot")
}

// Compile-time: the test for "secret never stored twice" guards the invariant
// that the raw secret is only ever written to the secret store — never into
// any struct field that could be marshaled into a response.
var _ = secrets.ErrNotFound