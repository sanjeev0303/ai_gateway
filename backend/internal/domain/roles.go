package domain

// Organization roles. RoleAdmin and RoleMember are declared alongside the
// other shared status/role values in model.go; this block adds the roles that
// are specific to the identity milestone. The set mirrors the CHECK in
// organization_members and invitations; the database remains the source of
// truth.
const (
	RoleOwner        = "owner"
	RoleAuditor      = "auditor"
	RoleBillingAdmin = "billing_admin"
	// RoleService is a pseudo-role for machine identities: it is never stored
	// in the database and is only used to grant service-account actors their
	// fixed permission set.
	RoleService = "service"
)

// Permissions are the coarse-grained capabilities granted to organization
// roles. Individual operations map onto one permission.
const (
	PermissionOrgRead           = "org.read"
	PermissionOrgUpdate         = "org.update"
	PermissionOrgManage         = "org.manage"
	PermissionMembersRead       = "members.read"
	PermissionMembersManage     = "members.manage"
	PermissionTeamsRead         = "teams.read"
	PermissionTeamsManage       = "teams.manage"
	PermissionProjectsRead      = "projects.read"
	PermissionProjectsManage    = "projects.manage"
	PermissionProvidersRead     = "providers.read"
	PermissionProvidersManage   = "providers.manage"
	PermissionCredentialsRead   = "credentials.read"
	PermissionCredentialsManage = "credentials.manage"
	PermissionUsageRead         = "usage.read"
	PermissionAuditRead         = "audit.read"
	PermissionBillingRead       = "billing.read"
	PermissionBillingManage     = "billing.manage"
	PermissionAlertsRead        = "alerts.read"
	PermissionAlertsManage      = "alerts.manage"
	// PermissionSecurityEventsRead gates the platform-originated event feed.
	// There is no manage variant: security_events is an append-only ledger
	// written only by the components that detect a condition, so there is no
	// caller-side mutation to authorize.
	PermissionSecurityEventsRead = "security_events.read"
	PermissionGuardrailsRead     = "guardrails.read"
	PermissionGuardrailsManage   = "guardrails.manage"
)

// rolePermissions maps each role to the permissions it grants.
//
//	owner          everything
//	admin          everything except org/billing management
//	auditor        read-only across the organization
//	billing_admin  reads plus billing management
//	member         day-to-day reads and usage
//	service        machine identity: reads available to automation
var rolePermissions = map[string]map[string]bool{
	RoleOwner: allPermissions(),
	RoleAdmin: {
		PermissionOrgRead:            true,
		PermissionMembersRead:        true,
		PermissionMembersManage:      true,
		PermissionTeamsRead:          true,
		PermissionTeamsManage:        true,
		PermissionProjectsRead:       true,
		PermissionProjectsManage:     true,
		PermissionProvidersRead:      true,
		PermissionProvidersManage:    true,
		PermissionCredentialsRead:    true,
		PermissionCredentialsManage:  true,
		PermissionUsageRead:          true,
		PermissionAuditRead:          true,
		PermissionAlertsRead:         true,
		PermissionAlertsManage:       true,
		PermissionGuardrailsRead:     true,
		PermissionGuardrailsManage:   true,
		PermissionSecurityEventsRead: true,
		PermissionBillingRead:        true,
	},
	RoleAuditor: {
		PermissionOrgRead:            true,
		PermissionMembersRead:        true,
		PermissionTeamsRead:          true,
		PermissionProjectsRead:       true,
		PermissionProvidersRead:      true,
		PermissionCredentialsRead:    true,
		PermissionUsageRead:          true,
		PermissionAuditRead:          true,
		PermissionAlertsRead:         true,
		PermissionGuardrailsRead:     true,
		PermissionSecurityEventsRead: true,
		PermissionBillingRead:        true,
	},
	RoleBillingAdmin: {
		PermissionOrgRead:       true,
		PermissionMembersRead:   true,
		PermissionTeamsRead:     true,
		PermissionProjectsRead:  true,
		PermissionUsageRead:     true,
		PermissionBillingRead:   true,
		PermissionBillingManage: true,
	},
	RoleMember: {
		PermissionOrgRead:      true,
		PermissionMembersRead:  true,
		PermissionTeamsRead:    true,
		PermissionProjectsRead: true,
		PermissionUsageRead:    true,
	},
	RoleService: {
		PermissionOrgRead:     true,
		PermissionUsageRead:   true,
		PermissionAuditRead:   true,
		PermissionBillingRead: true,
	},
}

func allPermissions() map[string]bool {
	return map[string]bool{
		PermissionOrgRead:            true,
		PermissionOrgUpdate:          true,
		PermissionOrgManage:          true,
		PermissionMembersRead:        true,
		PermissionMembersManage:      true,
		PermissionTeamsRead:          true,
		PermissionTeamsManage:        true,
		PermissionProjectsRead:       true,
		PermissionProjectsManage:     true,
		PermissionProvidersRead:      true,
		PermissionProvidersManage:    true,
		PermissionCredentialsRead:    true,
		PermissionCredentialsManage:  true,
		PermissionUsageRead:          true,
		PermissionAuditRead:          true,
		PermissionBillingRead:        true,
		PermissionBillingManage:      true,
		PermissionAlertsRead:         true,
		PermissionAlertsManage:       true,
		PermissionGuardrailsRead:     true,
		PermissionGuardrailsManage:   true,
		PermissionSecurityEventsRead: true,
	}
}

// RoleHasPermission reports whether role may exercise perm. Unknown roles are
// denied everything.
func RoleHasPermission(role, perm string) bool {
	return rolePermissions[role][perm]
}

// IsOrgRole reports whether role is a storable organization role (as opposed
// to the pseudo-role used for machine identities).
func IsOrgRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember, RoleAuditor, RoleBillingAdmin:
		return true
	default:
		return false
	}
}
