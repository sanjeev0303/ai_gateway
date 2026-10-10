export type Role = "owner" | "admin" | "auditor" | "billing_admin" | "member" | "service";

export type Permission =
  | "org.read"
  | "org.update"
  | "org.manage"
  | "members.read"
  | "members.manage"
  | "teams.read"
  | "teams.manage"
  | "projects.read"
  | "projects.manage"
  | "providers.read"
  | "providers.manage"
  | "credentials.read"
  | "credentials.manage"
  | "usage.read"
  | "audit.read"
  | "billing.read"
  | "billing.manage"
  | "alerts.read"
  | "alerts.manage"
  | "guardrails.read"
  | "guardrails.manage"
  | "security_events.read"
  | "policies.read"
  | "policies.manage"
  | "model_access.read"
  | "model_access.manage"
  | "budgets.read"
  | "budgets.manage"
  | "gateway_credentials.read"
  | "gateway_credentials.manage"
  | "service_accounts.read"
  | "service_accounts.manage"
  | "provider_accounts.read"
  | "provider_accounts.manage"
  | "provider_credentials.read"
  | "provider_credentials.manage";

const rolePermissions: Record<Role, Permission[]> = {
  owner: [
    "org.read", "org.update", "org.manage",
    "members.read", "members.manage",
    "teams.read", "teams.manage",
    "projects.read", "projects.manage",
    "providers.read", "providers.manage",
    "credentials.read", "credentials.manage",
    "usage.read",
    "audit.read",
    "billing.read", "billing.manage",
    "alerts.read", "alerts.manage",
    "guardrails.read", "guardrails.manage",
    "security_events.read",
    "policies.read", "policies.manage",
    "model_access.read", "model_access.manage",
    "budgets.read", "budgets.manage",
    "gateway_credentials.read", "gateway_credentials.manage",
    "service_accounts.read", "service_accounts.manage",
    "provider_accounts.read", "provider_accounts.manage",
    "provider_credentials.read", "provider_credentials.manage",
  ],
  admin: [
    "org.read",
    "members.read", "members.manage",
    "teams.read", "teams.manage",
    "projects.read", "projects.manage",
    "providers.read", "providers.manage",
    "credentials.read", "credentials.manage",
    "usage.read",
    "audit.read",
    "billing.read",
    "alerts.read", "alerts.manage",
    "guardrails.read", "guardrails.manage",
    "security_events.read",
    "policies.read", "policies.manage",
    "model_access.read", "model_access.manage",
    "budgets.read", "budgets.manage",
    "gateway_credentials.read", "gateway_credentials.manage",
    "service_accounts.read", "service_accounts.manage",
    "provider_accounts.read", "provider_accounts.manage",
    "provider_credentials.read", "provider_credentials.manage",
  ],
  auditor: [
    "org.read",
    "members.read",
    "teams.read",
    "projects.read",
    "providers.read",
    "credentials.read",
    "usage.read",
    "audit.read",
    "alerts.read",
    "guardrails.read",
    "security_events.read",
    "billing.read",
    "policies.read",
    "model_access.read",
    "budgets.read",
    "gateway_credentials.read",
    "service_accounts.read",
    "provider_accounts.read",
    "provider_credentials.read",
  ],
  billing_admin: [
    "org.read",
    "members.read",
    "teams.read",
    "projects.read",
    "usage.read",
    "billing.read", "billing.manage",
  ],
  member: [
    "org.read",
    "members.read",
    "teams.read",
    "projects.read",
    "usage.read",
  ],
  service: [
    "org.read",
    "usage.read",
    "audit.read",
    "billing.read",
  ],
};

export function hasPermission(role: Role, permission: Permission): boolean {
  return rolePermissions[role]?.includes(permission) ?? false;
}

export function hasAnyPermission(role: Role, permissions: Permission[]): boolean {
  return permissions.some((p) => hasPermission(role, p));
}

export function hasAllPermissions(role: Role, permissions: Permission[]): boolean {
  return permissions.every((p) => hasPermission(role, p));
}

export function getPermissionsForRole(role: Role): Permission[] {
  return rolePermissions[role] ?? [];
}

export function getRolesWithPermission(permission: Permission): Role[] {
  return Object.entries(rolePermissions)
    .filter(([, perms]) => perms.includes(permission))
    .map(([role]) => role as Role);
}

export const PERMISSION_GROUPS = {
  organization: ["org.read", "org.update", "org.manage"] as Permission[],
  members: ["members.read", "members.manage"] as Permission[],
  teams: ["teams.read", "teams.manage"] as Permission[],
  projects: ["projects.read", "projects.manage"] as Permission[],
  providers: ["providers.read", "providers.manage"] as Permission[],
  credentials: ["credentials.read", "credentials.manage"] as Permission[],
  usage: ["usage.read"] as Permission[],
  audit: ["audit.read"] as Permission[],
  billing: ["billing.read", "billing.manage"] as Permission[],
  alerts: ["alerts.read", "alerts.manage"] as Permission[],
  guardrails: ["guardrails.read", "guardrails.manage"] as Permission[],
  security_events: ["security_events.read"] as Permission[],
  policies: ["policies.read", "policies.manage"] as Permission[],
  model_access: ["model_access.read", "model_access.manage"] as Permission[],
  budgets: ["budgets.read", "budgets.manage"] as Permission[],
  gateway_credentials: ["gateway_credentials.read", "gateway_credentials.manage"] as Permission[],
  service_accounts: ["service_accounts.read", "service_accounts.manage"] as Permission[],
  provider_accounts: ["provider_accounts.read", "provider_accounts.manage"] as Permission[],
  provider_credentials: ["provider_credentials.read", "provider_credentials.manage"] as Permission[],
} as const;

export type PermissionGroup = keyof typeof PERMISSION_GROUPS;