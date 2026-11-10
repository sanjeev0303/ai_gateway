import type { Metadata } from "next";

import type { AdminAlert, AdminPage } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { PageTabs } from "../_components/page-tabs";
import { hrefWith, PAGE_SIZE, pageFrom, param, serviceLabel, type SearchParams } from "../_lib";
import { AlertActionsCell, type AlertActionRow } from "./alert-actions";

export const metadata: Metadata = { title: "Alerts · Platform console" };

interface AlertRow extends AlertActionRow {
  title: string;
  detail: string;
  severity: string;
  severityTone: StatusTone;
  service: string;
  status: string;
  statusTone: StatusTone;
  opened: string;
  openedAt: string;
  notes: string;
}

const COLUMNS: Column<AlertRow>[] = [
  { key: "severity", header: "Severity", primary: "severity", tone: "severityTone" },
  { key: "title", header: "Alert", primary: "title", secondary: "detail", grow: true },
  { key: "service", header: "Service", primary: "service" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", secondary: "notes" },
  { key: "opened", header: "Opened", primary: "opened", sort: "openedAt" },
];

const STATUS_TONE: Record<string, StatusTone> = { open: "danger", acknowledged: "warning", resolved: "neutral" };

const TABS = [
  { value: "live", label: "Live" },
  { value: "resolved", label: "Resolved" },
  { value: "all", label: "All" },
];

function toRow(alert: AdminAlert): AlertRow {
  const notes: string[] = [];
  if (alert.acknowledged_by) notes.push(`by ${alert.acknowledged_by}`);
  if (alert.resolved_at) notes.push(`resolved ${formatRelative(alert.resolved_at)}`);
  notes.push(alert.emailed ? "emailed" : "not emailed");
  return {
    id: alert.id,
    open: alert.status === "open",
    title: alert.title,
    detail: alert.detail,
    severity: alert.severity === "critical" ? "Critical" : "Warning",
    severityTone: alert.severity === "critical" ? "danger" : "warning",
    service: serviceLabel(alert.service),
    status: alert.status.charAt(0).toUpperCase() + alert.status.slice(1),
    statusTone: STATUS_TONE[alert.status] ?? "neutral",
    opened: formatRelative(alert.opened_at),
    openedAt: alert.opened_at,
    notes: notes.join(", "),
  };
}

export default async function AlertsPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const view = param(params, "view") || "live";
  const { page, offset } = pageFrom(params);

  let result: AdminPage<AdminAlert>;
  try {
    const { admin } = await getApiClient();
    result = await admin.alerts({ status: view === "all" ? undefined : view, limit: PAGE_SIZE, offset });
  } catch {
    return (
      <PageShell title="Alerts">
        <ErrorState description="Alerts could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }
  const hasMore = offset + result.items.length < result.total;

  return (
    <PageShell
      title="Alerts"
      description={`${formatNumber(result.total)} ${view === "live" ? "live" : view === "resolved" ? "resolved" : "in total"}. Raised by the worker when a service is down, runs hot, logs a burst of errors or falls behind; emailed to platform admins and resolved when the problem clears.`}
      actions={
        <PageTabs
          label="Alert status"
          current={view}
          tabs={TABS.map((t) => ({ ...t, href: hrefWith("/alerts", params, { view: t.value === "live" ? null : t.value, page: null }) }))}
        />
      }
    >
      <DataTable
        columns={COLUMNS}
        rows={result.items.map(toRow)}
        rowKey="id"
        ActionsComponent={AlertActionsCell}
        emptyMessage={view === "live" ? "Nothing needs attention. Every service is up and within its limits." : "No alerts here."}
        serverPaginated
        currentPage={page - 1}
        hasMore={hasMore}
        prevHref={page > 1 ? hrefWith("/alerts", params, { page: page - 1 }) : undefined}
        nextHref={hasMore ? hrefWith("/alerts", params, { page: page + 1 }) : undefined}
      />
    </PageShell>
  );
}
