import type { Metadata } from "next";

import type { AdminAuditEntry, AdminSecurityEvent, AdminTenantAuditEntry, ApiClient } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { ListFilters } from "../_components/list-filters";
import { PageTabs } from "../_components/page-tabs";
import { actionLabel, hrefWith, PAGE_SIZE, pageFrom, param, type SearchParams } from "../_lib";

export const metadata: Metadata = { title: "Audit · Platform console" };

const TABS = [
  { value: "platform", label: "Platform admins" },
  { value: "organizations", label: "Organizations" },
  { value: "security", label: "Security events" },
];

// ---- platform admin actions -------------------------------------------------

interface PlatformRow {
  id: string;
  occurredAt: string;
  action: string;
  detail: string;
  target: string;
  href: string;
  actor: string;
  reason: string;
}

const PLATFORM_COLUMNS: Column<PlatformRow>[] = [
  { key: "when", header: "When", primary: "occurredAt", format: "dateTime" },
  { key: "action", header: "Action", primary: "action", secondary: "detail" },
  { key: "target", header: "Target", primary: "target", href: "href" },
  { key: "actor", header: "Platform admin", primary: "actor" },
  { key: "reason", header: "Reason", primary: "reason", grow: true },
];

function platformRow(entry: AdminAuditEntry): PlatformRow {
  let target = entry.target_type;
  let href = "";
  if (entry.organization_id) {
    target = entry.organization_name || "Deleted organization";
    href = `/organizations/${entry.organization_id}`;
  } else if (entry.target_type === "user") {
    target = "User";
    href = entry.target_id ? `/users?search=${entry.target_id}` : "";
  } else if (entry.target_type === "model") {
    target = "Model catalog";
    href = "/models";
  } else if (entry.target_type === "alert") {
    target = "Alert";
    href = "/alerts?view=all";
  }
  const { plan_code: plan, revoked } = entry.detail as { plan_code?: string; revoked?: number };
  return {
    id: entry.id,
    occurredAt: entry.created_at,
    action: actionLabel(entry.action),
    detail: plan ? `To ${plan}` : typeof revoked === "number" ? `${revoked} keys` : "",
    target,
    href,
    actor: entry.actor_email,
    reason: entry.reason,
  };
}

// ---- organizations' own audit logs ------------------------------------------

interface TenantRow {
  id: string;
  occurredAt: string;
  action: string;
  resource: string;
  organization: string;
  href: string;
  actor: string;
  origin: string;
}

const TENANT_COLUMNS: Column<TenantRow>[] = [
  { key: "when", header: "When", primary: "occurredAt", format: "dateTime" },
  { key: "action", header: "Action", primary: "action", secondary: "resource", grow: true },
  { key: "org", header: "Organization", primary: "organization", href: "href" },
  { key: "actor", header: "Actor", primary: "actor" },
  { key: "origin", header: "IP address", primary: "origin" },
];

function tenantRow(entry: AdminTenantAuditEntry): TenantRow {
  return {
    id: entry.id,
    occurredAt: entry.created_at,
    action: entry.action,
    resource: entry.resource_id ? `${entry.resource_type} ${entry.resource_id.slice(0, 8)}…` : entry.resource_type,
    organization: entry.organization_name,
    href: `/organizations/${entry.organization_id}`,
    actor: entry.actor_email || "System or service account",
    origin: entry.ip_address || "—",
  };
}

// ---- security events --------------------------------------------------------

interface SecurityRow {
  id: string;
  occurredAt: string;
  severity: string;
  severityTone: StatusTone;
  title: string;
  category: string;
  organization: string;
  href: string;
  source: string;
}

const SECURITY_COLUMNS: Column<SecurityRow>[] = [
  { key: "when", header: "When", primary: "occurredAt", format: "dateTime" },
  { key: "severity", header: "Severity", primary: "severity", tone: "severityTone" },
  { key: "title", header: "Event", primary: "title", secondary: "category", grow: true },
  { key: "org", header: "Organization", primary: "organization", href: "href" },
  { key: "source", header: "Source", primary: "source" },
];

const SEVERITY_TONE: Record<string, StatusTone> = { critical: "danger", warning: "warning", info: "info" };

function securityRow(event: AdminSecurityEvent): SecurityRow {
  return {
    id: event.id,
    occurredAt: event.created_at,
    severity: event.severity.charAt(0).toUpperCase() + event.severity.slice(1),
    severityTone: SEVERITY_TONE[event.severity] ?? "neutral",
    title: event.title,
    category: event.category.replace(/_/g, " "),
    organization: event.organization_name,
    href: `/organizations/${event.organization_id}`,
    source: event.source,
  };
}

export default async function AuditPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const view = TABS.some((t) => t.value === param(params, "view")) ? param(params, "view") : "platform";
  const { page, offset } = pageFrom(params);
  const tabs = (
    <PageTabs
      label="Audit trail"
      current={view}
      tabs={TABS.map((t) => ({ ...t, href: hrefWith("/audit", {}, { view: t.value === "platform" ? null : t.value }) }))}
    />
  );
  const pager = (hasMore: boolean) => ({
    serverPaginated: true as const,
    currentPage: page - 1,
    hasMore,
    prevHref: page > 1 ? hrefWith("/audit", params, { page: page - 1 }) : undefined,
    nextHref: hasMore ? hrefWith("/audit", params, { page: page + 1 }) : undefined,
  });

  // Fetch first, render after: JSX built inside try/catch would not have its
  // render errors caught there.
  let tenant: Awaited<ReturnType<ApiClient["admin"]["tenantAudit"]>> | null = null;
  let security: Awaited<ReturnType<ApiClient["admin"]["securityEvents"]>> | null = null;
  let platform: Awaited<ReturnType<ApiClient["admin"]["audit"]>> | null = null;
  const action = param(params, "search");
  const severity = param(params, "severity");
  try {
    const { admin } = await getApiClient();
    if (view === "organizations") tenant = await admin.tenantAudit({ action, limit: PAGE_SIZE, offset });
    else if (view === "security") security = await admin.securityEvents({ severity, limit: PAGE_SIZE, offset });
    else platform = await admin.audit({ limit: PAGE_SIZE, offset });
  } catch {
    return (
      <PageShell title="Audit" actions={tabs}>
        <ErrorState description="The audit trail could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  if (tenant) {
    return (
      <PageShell
        title="Audit"
        description={`${formatNumber(tenant.total)} entries from every organization's own audit log, newest first: sign-ins, key changes, policy edits and more.`}
        actions={tabs}
      >
        <DataTable
          columns={TENANT_COLUMNS}
          rows={tenant.items.map(tenantRow)}
          rowKey="id"
          toolbar={<ListFilters searchLabel="Filter by action" placeholder="Action, e.g. auth.login…" />}
          emptyMessage={action ? "No entry starts with that action." : "No organization has recorded an audit entry yet."}
          {...pager(offset + tenant.items.length < tenant.total)}
        />
      </PageShell>
    );
  }

  if (security) {
    return (
      <PageShell
        title="Audit"
        description={`${formatNumber(security.total)} security events across every organization: provider incidents and pipeline failures.`}
        actions={tabs}
      >
        <DataTable
          columns={SECURITY_COLUMNS}
          rows={security.items.map(securityRow)}
          rowKey="id"
          toolbar={
            <PageTabs
              label="Severity"
              current={severity || "all"}
              tabs={[
                { value: "all", label: "Any severity" },
                { value: "critical", label: "Critical" },
                { value: "warning", label: "Warning" },
                { value: "info", label: "Info" },
              ].map((t) => ({ ...t, href: hrefWith("/audit", params, { severity: t.value === "all" ? null : t.value, page: null }) }))}
            />
          }
          emptyMessage="No security event recorded."
          {...pager(offset + security.items.length < security.total)}
        />
      </PageShell>
    );
  }

  const entries = platform?.items ?? [];
  const total = platform?.total ?? 0;
  return (
    <PageShell
      title="Audit"
      description={`${formatNumber(total)} platform admin actions, newest first. Append-only: no one can edit or delete an entry. Actions on an organization are also in its own audit log.`}
      actions={tabs}
    >
      <DataTable
        columns={PLATFORM_COLUMNS}
        rows={entries.map(platformRow)}
        rowKey="id"
        emptyMessage="No platform admin has taken an action yet."
        {...pager(offset + entries.length < total)}
      />
    </PageShell>
  );
}
