import type { Metadata } from "next";

import type { AuditLog } from "@ai-governance/api-client";
import {
  DataTable,
  PageShell,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getApiClient } from "@/lib/session";
import { AuditLogActions } from "./client";
import { AuditFilters } from "./filters";
import { AuditActionsCell } from "./actions-cell";
import { ExportAuditLogsButton } from "./export-button";

export const metadata: Metadata = {
  title: "Audit logs · Sentinel",
  description: "Append-only record of every actor, action, and resource touched in the console.",
};

export interface AuditRow {
  id: string;
  action: string;
  actor: string;
  actorId: string;
  resource: string;
  resourceId: string;
  changes: string;
  origin: string;
  occurredAt: string;
  rawAlert: AuditLog;
}

/**
 * Audit records are append-only, so this view is deliberately read-only: there
 * is no control here that could edit or remove a record, matching the API.
 */

/** Actor column shows which kind of actor acted, with the id muted beneath. */
function actorLabel(record: AuditLog): string {
  return record.actor_type === "service_account" ? "Service account" : "User";
}

/** Counts top-level keys that actually differ between `before` and `after`. */
function changedKeys(before: AuditLog["before"], after: AuditLog["after"]): number {
  const keys = new Set([...Object.keys(before ?? {}), ...Object.keys(after ?? {})]);
  let changed = 0;
  for (const key of keys) {
    if (JSON.stringify(before?.[key]) !== JSON.stringify(after?.[key])) changed += 1;
  }
  return changed;
}

function toRow(record: AuditLog): AuditRow {
  const changed = changedKeys(record.before, record.after);

  return {
    id: record.id,
    action: record.action,
    actor: actorLabel(record),
    actorId: record.actor_user_id ?? record.actor_service_account_id ?? "",
    resource: record.resource_type,
    resourceId: record.resource_id ?? "",
    changes:
      changed === 0
        ? "No field-level change recorded"
        : `${changed} field${changed === 1 ? "" : "s"} changed`,
    origin: [record.ip_address, record.request_id].filter(Boolean).join(" · ") || "—",
    occurredAt: record.created_at,
    rawAlert: record,
  };
}

const columns: readonly Column<AuditRow>[] = [
  { key: "action", header: "Action", primary: "action", secondary: "changes", sort: "action" },
  { key: "actor", header: "Actor", primary: "actor", secondary: "actorId", sort: "actor" },
  {
    key: "resource",
    header: "Resource",
    primary: "resource",
    secondary: "resourceId",
    sort: "resource",
  },
  { key: "origin", header: "Origin", primary: "origin", sort: "origin" },
  {
    key: "occurredAt",
    header: "When",
    primary: "occurredAt",
    format: "relative",
    align: "right",
    sort: "occurredAt",
  },
];

async function loadAuditLogs(params: { [key: string]: string | string[] | undefined }) {
  const { auditLogs } = await getApiClient();
  const limit = typeof params.limit === "string" ? parseInt(params.limit, 10) : 25;
  const offset = typeof params.offset === "string" ? parseInt(params.offset, 10) : 0;
  const res = await auditLogs.list({
    limit,
    offset,
    actor_id: typeof params.actor_id === "string" ? params.actor_id : undefined,
    action: typeof params.action === "string" ? params.action : undefined,
    resource_type: typeof params.resource_type === "string" ? params.resource_type : undefined,
  });
  return { ...res, limit, offset };
}

export default async function AuditLogsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const { items, has_more, limit, offset } = await loadAuditLogs(params);
  const rows = items.map(toRow);

  const currentPage = Math.floor(offset / limit);

  const buildPageUrl = (newOffset: number) => {
    const p = new URLSearchParams();
    if (typeof params.actor_id === "string") p.set("actor_id", params.actor_id);
    if (typeof params.action === "string") p.set("action", params.action);
    if (typeof params.resource_type === "string") p.set("resource_type", params.resource_type);
    p.set("limit", limit.toString());
    p.set("offset", newOffset.toString());
    return `/app/security/audit-logs?${p.toString()}`;
  };

  const nextHref = buildPageUrl(offset + limit);
  const prevHref = buildPageUrl(Math.max(0, offset - limit));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Audit logs"
        description="Append-only record of every actor, action, and resource touched in the console."
        actions={<ExportAuditLogsButton rows={rows} />}
      >
        <AuditFilters />
        <Card>
          <CardHeader>
            <CardTitle>Actor and action history</CardTitle>
            <CardDescription>
              Newest first, scoped to your organization. Records can never be edited or deleted.
            </CardDescription>
          </CardHeader>
          <CardContent>
              <DataTable
              columns={columns}
              rows={rows}
              rowKey="id"
              pageSize={limit}
              emptyMessage="No audit records yet"
              ActionsComponent={AuditActionsCell}
              serverPaginated
              hasMore={has_more}
              currentPage={currentPage}
              nextHref={nextHref}
              prevHref={prevHref}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}