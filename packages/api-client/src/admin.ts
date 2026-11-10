import type { components } from "@ai-governance/contracts";

import type { RequestOptions, Transport } from "./client";

type Schemas = components["schemas"];

/**
 * The super-admin console (`/v1/admin`). Every route answers 404 unless the
 * caller is a platform admin (their email is on the control API's
 * PLATFORM_ADMIN_EMAILS allowlist). Money is in micro-dollars throughout.
 */

export type AdminWindow = "24h" | "7d" | "30d" | "90d";

export type AdminMe = Schemas["AdminMe"];
export type AdminOverview = Schemas["AdminOverview"];
export type AdminUsageRow = Schemas["AdminUsageRow"];
export type AdminAuditEntry = Schemas["AdminAuditEntry"];
export type AdminOrganization = Schemas["AdminOrganization"];
export type AdminOrganizationDetail = Schemas["AdminOrganizationDetail"];
export type AdminUser = Schemas["AdminUser"];
export type AdminUsage = Schemas["AdminUsage"];
export type AdminBilling = Schemas["AdminBilling"];
export type AdminSystem = Schemas["AdminSystem"];
export type AdminServiceSample = Schemas["AdminServiceSample"];
export type AdminServices = Schemas["AdminServices"];
export type AdminAlert = Schemas["AdminAlert"];
export type AdminServiceLog = Schemas["AdminServiceLog"];
export type AdminTenantAuditEntry = Schemas["AdminTenantAuditEntry"];
export type AdminSecurityEvent = Schemas["AdminSecurityEvent"];
export type AdminRevenue = Schemas["AdminRevenue"];
export type AdminPayment = Schemas["AdminPayment"];

/** Every paged admin list has this shape. */
export interface AdminPage<TItem> {
  items: TItem[];
  total: number;
}

export interface AdminListQuery {
  search?: string;
  status?: string;
  window?: AdminWindow;
  /** 1 to 100; the server defaults to 25. */
  limit?: number;
  offset?: number;
}

export interface AdminModule {
  me(options?: RequestOptions): Promise<AdminMe>;
  overview(window?: AdminWindow, options?: RequestOptions): Promise<AdminOverview>;
  organizations(query?: AdminListQuery, options?: RequestOptions): Promise<AdminPage<AdminOrganization>>;
  organization(id: string, window?: AdminWindow, options?: RequestOptions): Promise<AdminOrganizationDetail>;
  suspendOrganization(id: string, reason: string, options?: RequestOptions): Promise<void>;
  reactivateOrganization(id: string, reason: string, options?: RequestOptions): Promise<void>;
  setOrganizationPlan(id: string, planCode: string, reason: string, options?: RequestOptions): Promise<void>;
  revokeOrganizationKeys(id: string, reason: string, options?: RequestOptions): Promise<Schemas["AdminRevokeKeysResponse"]>;
  users(query?: AdminListQuery, options?: RequestOptions): Promise<AdminPage<AdminUser>>;
  disableUser(id: string, reason: string, options?: RequestOptions): Promise<void>;
  enableUser(id: string, reason: string, options?: RequestOptions): Promise<void>;
  usage(window?: AdminWindow, options?: RequestOptions): Promise<AdminUsage>;
  billing(options?: RequestOptions): Promise<AdminBilling>;
  plans(options?: RequestOptions): Promise<Schemas["PlanListResponse"]>;
  system(options?: RequestOptions): Promise<AdminSystem>;
  models(options?: RequestOptions): Promise<Schemas["AdminModelListResponse"]>;
  setModelActive(id: string, active: boolean, reason: string, options?: RequestOptions): Promise<void>;
  audit(query?: { limit?: number; offset?: number }, options?: RequestOptions): Promise<AdminPage<AdminAuditEntry>>;
  tenantAudit(
    query?: { organization_id?: string; action?: string; limit?: number; offset?: number },
    options?: RequestOptions,
  ): Promise<AdminPage<AdminTenantAuditEntry>>;
  securityEvents(query?: { severity?: string; limit?: number; offset?: number }, options?: RequestOptions): Promise<AdminPage<AdminSecurityEvent>>;
  services(options?: RequestOptions): Promise<AdminServices>;
  /** status: "live" (open or acknowledged), a single status, or omitted for all. */
  alerts(query?: { status?: string; limit?: number; offset?: number }, options?: RequestOptions): Promise<AdminPage<AdminAlert>>;
  acknowledgeAlert(id: string, options?: RequestOptions): Promise<void>;
  logs(
    query?: { service?: string; level?: string; search?: string; limit?: number; offset?: number },
    options?: RequestOptions,
  ): Promise<AdminPage<AdminServiceLog>>;
  revenue(options?: RequestOptions): Promise<AdminRevenue>;
  payments(query?: { organization_id?: string; limit?: number; offset?: number }, options?: RequestOptions): Promise<AdminPage<AdminPayment>>;
}

export function createAdminModule(transport: Transport): AdminModule {
  const at = (path: string) => `/v1/admin/${path}`;
  const id = (value: string) => encodeURIComponent(value);

  return {
    me: (options) => transport.requestAt<AdminMe>(at("me"), options),
    overview: (window, options) =>
      transport.requestAt<AdminOverview>(at("overview"), { ...options, query: { window } }),
    organizations: (query, options) =>
      transport.requestAt<AdminPage<AdminOrganization>>(at("organizations"), { ...options, query: { ...query } }),
    organization: (orgId, window, options) =>
      transport.requestAt<AdminOrganizationDetail>(at(`organizations/${id(orgId)}`), { ...options, query: { window } }),
    suspendOrganization: (orgId, reason, options) =>
      transport.requestAt<void>(at(`organizations/${id(orgId)}/suspend`), { ...options, method: "POST", body: { reason } }),
    reactivateOrganization: (orgId, reason, options) =>
      transport.requestAt<void>(at(`organizations/${id(orgId)}/reactivate`), { ...options, method: "POST", body: { reason } }),
    setOrganizationPlan: (orgId, planCode, reason, options) =>
      transport.requestAt<void>(at(`organizations/${id(orgId)}/plan`), {
        ...options,
        method: "PUT",
        body: { plan_code: planCode, reason },
      }),
    revokeOrganizationKeys: (orgId, reason, options) =>
      transport.requestAt<Schemas["AdminRevokeKeysResponse"]>(at(`organizations/${id(orgId)}/revoke-keys`), {
        ...options,
        method: "POST",
        body: { reason },
      }),
    users: (query, options) =>
      transport.requestAt<AdminPage<AdminUser>>(at("users"), { ...options, query: { ...query } }),
    disableUser: (userId, reason, options) =>
      transport.requestAt<void>(at(`users/${id(userId)}/disable`), { ...options, method: "POST", body: { reason } }),
    enableUser: (userId, reason, options) =>
      transport.requestAt<void>(at(`users/${id(userId)}/enable`), { ...options, method: "POST", body: { reason } }),
    usage: (window, options) => transport.requestAt<AdminUsage>(at("usage"), { ...options, query: { window } }),
    billing: (options) => transport.requestAt<AdminBilling>(at("billing"), options),
    plans: (options) => transport.requestAt<Schemas["PlanListResponse"]>(at("plans"), options),
    system: (options) => transport.requestAt<AdminSystem>(at("system"), options),
    models: (options) => transport.requestAt<Schemas["AdminModelListResponse"]>(at("models"), options),
    setModelActive: (modelId, active, reason, options) =>
      transport.requestAt<void>(at(`models/${id(modelId)}`), { ...options, method: "PATCH", body: { active, reason } }),
    audit: (query, options) =>
      transport.requestAt<AdminPage<AdminAuditEntry>>(at("audit"), { ...options, query: { ...query } }),
    tenantAudit: (query, options) =>
      transport.requestAt<AdminPage<AdminTenantAuditEntry>>(at("tenant-audit"), { ...options, query: { ...query } }),
    securityEvents: (query, options) =>
      transport.requestAt<AdminPage<AdminSecurityEvent>>(at("security-events"), { ...options, query: { ...query } }),
    services: (options) => transport.requestAt<AdminServices>(at("services"), options),
    alerts: (query, options) => transport.requestAt<AdminPage<AdminAlert>>(at("alerts"), { ...options, query: { ...query } }),
    acknowledgeAlert: (alertId, options) =>
      transport.requestAt<void>(at(`alerts/${id(alertId)}/acknowledge`), { ...options, method: "POST" }),
    logs: (query, options) => transport.requestAt<AdminPage<AdminServiceLog>>(at("logs"), { ...options, query: { ...query } }),
    revenue: (options) => transport.requestAt<AdminRevenue>(at("revenue"), options),
    payments: (query, options) =>
      transport.requestAt<AdminPage<AdminPayment>>(at("payments"), { ...options, query: { ...query } }),
  };
}
