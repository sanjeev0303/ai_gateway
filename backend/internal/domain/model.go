// Package domain defines the core entities of the AI governance platform.
//
// The domain types are plain data holders: they carry no persistence, RLS, or
// sqlc concerns. Repositories map between these types and the sqlc-generated
// DTOs in internal/infrastructure/postgres/sqlc, so business logic never
// imports the generated database structs.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status and role values. These mirror the CHECK constraints enforced in the
// database; the database remains the source of truth.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

const (
	OrgStatusActive    = "active"
	OrgStatusSuspended = "suspended"
)

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

const (
	ProjectStatusActive   = "active"
	ProjectStatusArchived = "archived"
)

// NewOrganization carries the caller-supplied fields for creating an
// organization. Status is defaulted by the database to aigw 'active'.
type NewOrganization struct {
	Name string
	Slug string
}

// Organization is a tenant root. Global table: a small set of well-known
// organizations, never tenant-scoped.
type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUser carries the caller-supplied fields for creating a user.
type NewUser struct {
	Email       string
	DisplayName string
}

// User is a global identity that may belong to many organizations.
type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

// NewProject carries the caller-supplied fields for creating a project
// inside an organization.
type NewProject struct {
	Name        string
	Slug        string
	Description string
}

// Project groups usage and scoped access within an organization.
type Project struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Slug           string
	Description    string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewTeam carries the caller-supplied fields for creating a team.
type NewTeam struct {
	Name        string
	Description string
}

// Team is a tenant-scoped group of users.
type Team struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OrganizationMember is one row of the organization_members join table.
type OrganizationMember struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
	CreatedAt      time.Time
}

// TeamMember is one row of the team_members join table.
type TeamMember struct {
	TeamID         uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
	CreatedAt      time.Time
}

// OrgMember is an organization member with the user's profile attached.
type OrgMember struct {
	User        User
	Role        string
	MemberSince time.Time
}

// TeamMemberView is a team member with the user's profile attached.
type TeamMemberView struct {
	User        User
	Role        string
	MemberSince time.Time
}

// UserOrganization is an organization a user belongs to, plus their role.
type UserOrganization struct {
	Organization Organization
	Role         string
}
