package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenKind classifies the bearer of an issued JWT.
type TokenKind string

const (
	TokenKindUser    TokenKind = "user"
	TokenKindService TokenKind = "service"
)

// TokenClaims is the framework-free payload of an issued access token. The
// JWT layer (infrastructure) serializes it; the application layer reads it
// back without importing any crypto library.
type TokenClaims struct {
	Subject        string
	Email          string
	DisplayName    string
	OrganizationID uuid.UUID
	Role           string
	Kind           TokenKind
	IssuedAt       time.Time
	ExpiresAt      time.Time
}

// Actor materializes the claims into an authorization principal.
func (c TokenClaims) Actor() Actor {
	actor := Actor{
		Kind:           cursorKind(c.Kind),
		UserID:         mustParseUUID(c.Subject),
		OrganizationID: c.OrganizationID,
		Role:           c.Role,
		DisplayName:    c.DisplayName,
	}
	return actor
}

func cursorKind(k TokenKind) ActorKind {
	if k == TokenKindService {
		return ActorKindService
	}
	return ActorKindUser
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
