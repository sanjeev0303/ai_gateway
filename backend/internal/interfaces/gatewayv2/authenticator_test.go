package gatewayv2

import (
	"context"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

func fixedClock() func() time.Time {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

func TestAuthenticate_RejectsEmptyKey(t *testing.T) {
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			t.Fatal("resolve must not be called for empty key")
			return domain.GatewayCredential{}, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "")
	requireErrIs(t, err, ErrInvalidCredential)
}

func TestAuthenticate_HasherErrorPropagates(t *testing.T) {
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			t.Fatal("resolve must not be called when hashing fails")
			return domain.GatewayCredential{}, nil
		},
	}
	a := NewAuthenticator(&stubHasher{err: errStub}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, errStub)
}

func TestAuthenticate_CacheHitSkipsRepo(t *testing.T) {
	cache := &stubCredentialCache{
		entries: map[string]*CredentialCacheEntry{
			"hash:key-1": {
				ID:             mustUUIDs[0].String(),
				OrganizationID: mustUUIDs[1].String(),
				SubjectType:    "service_account",
				SubjectID:      mustUUIDs[2].String(),
				Scopes:         []string{"gateway"},
				Status:         "active",
			},
		},
	}
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			t.Fatal("resolve must not be called on cache hit")
			return domain.GatewayCredential{}, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, cache)

	got, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
	if got.ID != mustUUIDs[0] {
		t.Fatalf("expected cached credential id %s, got %s", mustUUIDs[0], got.ID)
	}
	if repo.resolveHits != 0 {
		t.Fatalf("expected 0 repo hits, got %d", repo.resolveHits)
	}
}

func TestAuthenticate_CachedEntryMapsToGatewayCredential(t *testing.T) {
	expires := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	teamID := mustUUIDs[2]
	cache := &stubCredentialCache{
		entries: map[string]*CredentialCacheEntry{
			"hash:key-1": {
				ID:             mustUUIDs[0].String(),
				OrganizationID: mustUUIDs[1].String(),
				SubjectType:    "user",
				SubjectID:      mustUUIDs[2].String(),
				TeamID:         strPtr(teamID.String()),
				Scopes:         []string{"all"},
				Status:         "active",
				ExpiresAt:      &expires,
			},
		},
	}
	a := NewAuthenticator(&stubHasher{}, &stubGatewayCredRepo{resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
		t.Fatal("resolve must not be called on cache hit")
		return domain.GatewayCredential{}, nil
	}}, cache)
	a.clock = fixedClock()

	got, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
	if got.SubjectType != "user" {
		t.Fatalf("expected subject type user, got %s", got.SubjectType)
	}
	if got.SubjectID != mustUUIDs[2] {
		t.Fatalf("expected subject id %s, got %s", mustUUIDs[2], got.SubjectID)
	}
	if got.TeamID == nil || *got.TeamID != teamID {
		t.Fatalf("expected team id %s, got %v", teamID, got.TeamID)
	}
	if got.Scopes[0] != "all" {
		t.Fatalf("expected scope all, got %v", got.Scopes)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expires) {
		t.Fatalf("expected expiry %v, got %v", expires, got.ExpiresAt)
	}
}

func TestAuthenticate_InvalidCachedEntryFallsThroughToRepo(t *testing.T) {
	// revoked status in cache must force a DB lookup
	cache := &stubCredentialCache{
		entries: map[string]*CredentialCacheEntry{
			"hash:key-1": {
				ID:             mustUUIDs[0].String(),
				OrganizationID: mustUUIDs[1].String(),
				SubjectType:    "service_account",
				SubjectID:      mustUUIDs[2].String(),
				Scopes:         []string{"gateway"},
				Status:         "revoked",
			},
		},
	}
	var repo *stubGatewayCredRepo
	repo = &stubGatewayCredRepo{
		resolveFn: func(_ context.Context, keyHash string) (domain.GatewayCredential, error) {
			if keyHash != "hash:key-1" {
				t.Fatalf("expected hash hash:key-1, got %s", keyHash)
			}
			return activeGatewayCredential(), nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, cache)

	got, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
	if got.ID != mustUUIDs[0] {
		t.Fatalf("expected credential id %s, got %s", mustUUIDs[0], got.ID)
	}
	if repo.resolveHits != 1 {
		t.Fatalf("expected 1 repo hit, got %d", repo.resolveHits)
	}
}

func TestAuthenticate_NotFoundMapsToInvalid(t *testing.T) {
	a := NewAuthenticator(&stubHasher{}, &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		},
	}, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrInvalidCredential)
}

func TestAuthenticate_RepoErrorPropagates(t *testing.T) {
	a := NewAuthenticator(&stubHasher{}, &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return domain.GatewayCredential{}, errStub
		},
	}, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, errStub)
}

func TestAuthenticate_PopulatesCacheOnRepoHit(t *testing.T) {
	cache := &stubCredentialCache{}
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return activeGatewayCredential(), nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, cache)

	_, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
	if cache.lastSet == nil {
		t.Fatal("expected credential to be cached after repo hit")
	}
	if cache.lastSet.ID != mustUUIDs[0].String() {
		t.Fatalf("expected cached id %s, got %s", mustUUIDs[0], cache.lastSet.ID)
	}
	if cache.lastSet.Status != "active" {
		t.Fatalf("expected cached status active, got %s", cache.lastSet.Status)
	}
}

func TestAuthenticate_RejectsRevokedCredential(t *testing.T) {
	cache := &stubCredentialCache{}
	cred := activeGatewayCredential()
	cred.Status = domain.GatewayCredentialStatusRevoked
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, cache)

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrCredentialRevoked)
	if cache.lastSet != nil {
		t.Fatal("revoked credential must not be cached")
	}
}

func TestAuthenticate_RejectsExpiredStatus(t *testing.T) {
	cred := activeGatewayCredential()
	cred.Status = domain.GatewayCredentialStatusExpired
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrCredentialExpired)
}

func TestAuthenticate_RejectsExpiredByTime(t *testing.T) {
	past := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	cred := activeGatewayCredential()
	cred.ExpiresAt = &past
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})
	a.clock = fixedClock()

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrCredentialExpired)
}

func TestAuthenticate_AcceptsNotYetExpired(t *testing.T) {
	future := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	cred := activeGatewayCredential()
	cred.ExpiresAt = &future
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})
	a.clock = fixedClock()

	_, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
}

func TestAuthenticate_RejectsWrongScope(t *testing.T) {
	cred := activeGatewayCredential()
	cred.Scopes = []domain.GatewayCredentialScope{domain.GatewayCredentialScopeControl}
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrWrongScope)
}

func TestAuthenticate_AcceptsAllScope(t *testing.T) {
	cred := activeGatewayCredential()
	cred.Scopes = []domain.GatewayCredentialScope{domain.GatewayCredentialScopeAll}
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
}

func TestAuthenticate_RejectsInvalidStatus(t *testing.T) {
	cred := activeGatewayCredential()
	cred.Status = "suspended"
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	_, err := a.Authenticate(context.Background(), "key-1")
	requireErrIs(t, err, ErrInvalidCredential)
}

func TestAuthenticate_ConvertsTeamAndProject(t *testing.T) {
	teamID := mustUUIDs[1]
	projectID := mustUUIDs[2]
	cred := activeGatewayCredential()
	cred.TeamID = &teamID
	cred.ProjectID = &projectID
	repo := &stubGatewayCredRepo{
		resolveFn: func(context.Context, string) (domain.GatewayCredential, error) {
			return cred, nil
		},
	}
	a := NewAuthenticator(&stubHasher{}, repo, &stubCredentialCache{})

	got, err := a.Authenticate(context.Background(), "key-1")
	requireNoErr(t, err)
	if got.TeamID == nil || *got.TeamID != teamID {
		t.Fatalf("expected team id %s, got %v", teamID, got.TeamID)
	}
	if got.ProjectID == nil || *got.ProjectID != projectID {
		t.Fatalf("expected project id %s, got %v", projectID, got.ProjectID)
	}
}

func activeGatewayCredential() domain.GatewayCredential {
	return domain.GatewayCredential{
		ID:             mustUUIDs[0],
		OrganizationID: mustUUIDs[1],
		Name:           "test-cred",
		KeyPrefix:      "aigw_test",
		KeyHash:        "hash:key-1",
		SubjectType:    domain.SubjectTypeServiceAccount,
		SubjectID:      mustUUIDs[2],
		Scopes:         []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
		Status:         domain.GatewayCredentialStatusActive,
	}
}