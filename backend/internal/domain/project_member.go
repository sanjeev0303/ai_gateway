package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project membership roles, mirroring the CHECK in project_members.
const (
	ProjectMemberRoleAdmin  = "admin"
	ProjectMemberRoleMember = "member"
)

// ProjectMember is one row of the project_members join table.
type ProjectMember struct {
	ProjectID      uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
	CreatedAt      time.Time
}

// ProjectMemberView is a project member with the user's profile attached.
type ProjectMemberView struct {
	User        User
	Role        string
	MemberSince time.Time
}
