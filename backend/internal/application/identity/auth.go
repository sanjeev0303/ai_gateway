package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

const (
	// defaultRefreshTokenTTL applies when the deployment supplies no
	// REFRESH_TOKEN_TTL. Thirty days bounds the exposure of a stolen refresh
	// cookie; single-use rotation keeps any one live token's window short.
	defaultRefreshTokenTTL = 30 * 24 * time.Hour

	// Password length bounds. The minimum follows current guidance for
	// user-chosen secrets; the ceiling keeps a pathological input from
	// turning the KDF into a cheap denial-of-service lever, and matches the
	// bcrypt-style limit operators already expect.
	minPasswordLength = 12
	maxPasswordLength = 128

	// maxRegistrationAttempts bounds slug collisions during signup. The first
	// attempt uses the clean slug, later ones append a random suffix, so a
	// persistent failure needs three unrelated collisions.
	maxRegistrationAttempts = 3

	// refreshTokenBytes gives a 256-bit opaque credential. Only its digest is
	// stored, so the database never holds a value that can be replayed.
	refreshTokenBytes = 32
)

// Register creates an account and signs the user in. The user, password
// digest, organization, owner membership, and first refresh token commit
// together; only then is the access token minted, so a failure cannot leave a
// half-provisioned account or a user holding no session. orgName defaults to
// the user's display name.
func (s *Service) Register(ctx context.Context, email, displayName, orgName, password string) (Session, error) {
	email = normalizeEmail(email)
	if email == "" || !acceptablePassword(password) {
		return Session{}, repositories.ErrInvalid
	}
	// Hashing is deliberately outside the transaction: it is the slow part
	// and must not hold row locks open.
	encoded, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return Session{}, err
	}
	// The refresh token is minted before the write so its digest can join
	// the transaction. A slug collision rolls the transaction back whole,
	// which discards the token too, so one value is safely reused across
	// attempts.
	raw, err := randomHex(refreshTokenBytes)
	if err != nil {
		return Session{}, err
	}
	refreshHash, err := s.hasher.Hash(ctx, raw)
	if err != nil {
		return Session{}, err
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = emailLocalPart(email)
	}
	orgLabel := strings.TrimSpace(orgName)
	if orgLabel == "" {
		orgLabel = name
	}
	slug := slugify(orgLabel)
	if slug == "" {
		slug = "org"
	}
	var (
		user domain.User
		org  domain.Organization
	)
	for attempt := 0; attempt < maxRegistrationAttempts; attempt++ {
		candidate := slug
		if attempt > 0 {
			suffix, err := randomHex(2)
			if err != nil {
				return Session{}, err
			}
			candidate = slug + "-" + suffix
		}
		user, org, err = s.auth.Register(ctx, domain.Registration{
			User:                  domain.NewUser{Email: email, DisplayName: name},
			PasswordHash:          encoded,
			RefreshTokenHash:      refreshHash,
			RefreshTokenExpiresAt: s.now().Add(s.refreshTTL),
			OrganizationID:        uuid.New(),
			OrganizationName:      orgLabel,
			OrganizationSlug:      candidate,
		})
		if err == nil {
			break
		}
		if !errors.Is(err, repositories.ErrConflict) {
			return Session{}, err
		}
	}
	if err != nil {
		return Session{}, err
	}
	s.audit(ctx, domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         user.ID,
		OrganizationID: org.ID,
		Role:           domain.RoleOwner,
		DisplayName:    user.DisplayName,
	}, "user.register", "user", &user.ID, nil, nil)
	session, err := s.newSession(ctx, user, org.ID, domain.RoleOwner)
	if err != nil {
		return Session{}, err
	}
	session.RefreshToken = raw
	return session, nil
}

// Authenticate verifies a password and issues a session. Every rejection
// path costs the same and returns the same ErrUnauthorized, so an attacker
// cannot tell an unknown email from a wrong password.
func (s *Service) Authenticate(ctx context.Context, email, password string) (Session, error) {
	credential, err := s.auth.GetCredentialByEmail(ctx, normalizeEmail(email))
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return Session{}, err
	}
	// Fall back to the decoy when the account is unknown or has never set a
	// password, so the KDF still runs on a real digest.
	encoded := s.passwords.DecoyDigest()
	if err == nil && credential.PasswordHash != nil {
		encoded = *credential.PasswordHash
	}
	if err := s.passwords.Verify(ctx, encoded, password); err != nil {
		return Session{}, ErrUnauthorized
	}
	if credential.User.ID == uuid.Nil || credential.User.Status != domain.UserStatusActive {
		return Session{}, ErrUnauthorized
	}
	if err := s.auth.TouchLastLogin(ctx, credential.User.ID); err != nil {
		return Session{}, err
	}
	orgs, err := s.users.ListOrganizationsForUser(ctx, credential.User.ID)
	if err != nil {
		return Session{}, err
	}
	if len(orgs) == 0 {
		return Session{}, ErrUnauthorized
	}
	s.audit(ctx, domain.Actor{
		OrganizationID: orgs[0].Organization.ID,
		UserID:         credential.User.ID,
		Kind:           domain.ActorKindUser,
		Role:           orgs[0].Role,
	}, "auth.login_success", "user", &credential.User.ID, nil, nil)
	return s.issueSession(ctx, credential.User, orgs[0].Organization.ID, orgs[0].Role)
}

// Refresh rotates a refresh token. The presented token is retired and a new
// one issued in the same call, so a given token authorises at most one
// exchange.
func (s *Service) Refresh(ctx context.Context, presented string) (Session, error) {
	if presented == "" {
		return Session{}, ErrUnauthorized
	}
	hash, err := s.hasher.Hash(ctx, presented)
	if err != nil {
		return Session{}, err
	}
	token, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	// A retired token coming back means the value leaked and the legitimate
	// holder is no longer the only one who has it. Sign the user out
	// everywhere rather than just refusing this request.
	if token.RevokedAt != nil {
		if err := s.refreshTokens.RevokeAllForUser(ctx, token.UserID); err != nil {
			return Session{}, err
		}
		return Session{}, ErrUnauthorized
	}
	if !s.now().Before(token.ExpiresAt) {
		return Session{}, ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	if user.Status != domain.UserStatusActive {
		return Session{}, ErrUnauthorized
	}
	// Revoke before reissuing. Losing this race means another request already
	// rotated the token, and the loser must not mint a second session.
	if err := s.refreshTokens.Revoke(ctx, hash); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	orgs, err := s.users.ListOrganizationsForUser(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	if len(orgs) == 0 {
		return Session{}, ErrUnauthorized
	}
	return s.issueSession(ctx, user, orgs[0].Organization.ID, orgs[0].Role)
}

// RevokeRefresh retires one refresh token. An unknown or already-retired
// token is not an error: logout is idempotent, because a client that lost
// track of its own session still needs to end cleanly.
func (s *Service) RevokeRefresh(ctx context.Context, presented string) error {
	if presented == "" {
		return nil
	}
	hash, err := s.hasher.Hash(ctx, presented)
	if err != nil {
		return err
	}
	if err := s.refreshTokens.Revoke(ctx, hash); err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return err
	}
	return nil
}

// SwitchOrganization issues a session scoped to the requested organization if
// the user is an active member of that organization.
func (s *Service) SwitchOrganization(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (Session, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	member, err := s.members.Get(ctx, orgID, userID)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	return s.issueSession(ctx, user, orgID, member.Role)
}

// newSession mints the access token for an already-persisted user. It writes
// no refresh state, so a caller that has already committed one can attach the
// raw value without persisting a second row.
func (s *Service) newSession(ctx context.Context, user domain.User, orgID uuid.UUID, role string) (Session, error) {
	claims := domain.TokenClaims{
		Subject:        user.ID.String(),
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		OrganizationID: orgID,
		Role:           role,
		Kind:           domain.TokenKindUser,
	}
	access, err := s.issuer.Issue(ctx, claims)
	if err != nil {
		return Session{}, err
	}
	return Session{Token: access, Claims: claims}, nil
}

// issueSession mints the access token and persists the digest of a fresh
// refresh token. The raw value is returned once, for the caller to place in
// an HttpOnly cookie; only the digest reaches the database.
func (s *Service) issueSession(ctx context.Context, user domain.User, orgID uuid.UUID, role string) (Session, error) {
	session, err := s.newSession(ctx, user, orgID, role)
	if err != nil {
		return Session{}, err
	}
	raw, err := randomHex(refreshTokenBytes)
	if err != nil {
		return Session{}, err
	}
	hash, err := s.hasher.Hash(ctx, raw)
	if err != nil {
		return Session{}, err
	}
	if _, err := s.refreshTokens.Create(ctx, user.ID, hash, s.now().Add(s.refreshTTL)); err != nil {
		return Session{}, err
	}
	session.RefreshToken = raw
	return session, nil
}

func acceptablePassword(password string) bool {
	return len(password) >= minPasswordLength && len(password) <= maxPasswordLength
}
