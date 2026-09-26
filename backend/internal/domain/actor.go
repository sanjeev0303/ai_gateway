package domain

import "github.com/google/uuid"

// ActorKind classifies who is acting on the control plane.
type ActorKind string

const (
	ActorKindUser    ActorKind = "user"
	ActorKindService ActorKind = "service"
)

// Actor is the authenticated principal of a control-plane call. It is derived
// from a verified JWT (user) or API key (machine identity) before any service
// method runs, so every method can authorize via RequirePermission.
type Actor struct {
	Kind           ActorKind
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	// DisplayName lets audit logs and responses identify the actor without a
	// lookup.
	DisplayName string
}

// Can reports whether the actor's role grants perm.
func (a Actor) Can(perm string) bool {
	return RoleHasPermission(a.Role, perm)
}

// ServiceActor builds the fixed-permission actor for machine identities. The
// service pseudo-role is never persisted; it only shapes permissions.
func ServiceActor(orgID, serviceAccountID uuid.UUID, displayName string) Actor {
	return Actor{
		Kind:           ActorKindService,
		UserID:         serviceAccountID,
		OrganizationID: orgID,
		Role:           RoleService,
		DisplayName:    displayName,
	}
}
