import type { Metadata } from "next";

import type { AdminPage, AdminServiceLog } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { ListFilters } from "../_components/list-filters";
import { PageTabs } from "../_components/page-tabs";
import { hrefWith, PAGE_SIZE, pageFrom, param, serviceLabel, type SearchParams } from "../_lib";

export const metadata: Metadata = { title: "Logs · Platform console" };

const SERVICES = ["gateway-v2", "control-api", "worker"];

interface LogRow {
  id: string;
  at: string;
  level: string;
  levelTone: StatusTone;
  service: string;
  message: string;
  attrs: string;
}

const COLUMNS: Column<LogRow>[] = [
  { key: "at", header: "Time", primary: "at", format: "dateTime" },
  { key: "level", header: "Level", primary: "level", tone: "levelTone" },
  { key: "service", header: "Service", primary: "service" },
  { key: "message", header: "Message", primary: "message", secondary: "attrs", grow: true },
];

/** The attributes as one readable line: key=value, the noisy ones left out. */
function attrLine(attrs: Record<string, unknown>): string {
  return Object.entries(attrs)
    .filter(([key]) => key !== "time" && key !== "level" && key !== "msg")
    .map(([key, value]) => `${key}=${typeof value === "string" ? value : JSON.stringify(value)}`)
    .join("  ");
}

export default async function LogsPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const { page, offset } = pageFrom(params);
  const service = param(params, "service");
  const level = param(params, "level");
  const search = param(params, "search");

  let result: AdminPage<AdminServiceLog>;
  try {
    const { admin } = await getApiClient();
    result = await admin.logs({ service, level, search, limit: PAGE_SIZE, offset });
  } catch {
    return (
      <PageShell title="Logs">
        <ErrorState description="Logs could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const rows: LogRow[] = result.items.map((log) => ({
    id: String(log.id),
    at: log.logged_at,
    level: log.level === "ERROR" ? "Error" : "Warning",
    levelTone: log.level === "ERROR" ? "danger" : "warning",
    service: serviceLabel(log.service),
    message: log.message,
    attrs: attrLine(log.attrs),
  }));
  const filtered = service !== "" || level !== "" || search !== "";
  const hasMore = offset + result.items.length < result.total;

  return (
    <PageShell
      title="Logs"
      description={`${formatNumber(result.total)} ${filtered ? "matching" : "warnings and errors"} from every service, newest first. Kept for 14 days; secrets are redacted before they are stored.`}
      actions={
        <PageTabs
          label="Service"
          current={service || "all"}
          tabs={[{ value: "all", label: "All services" }, ...SERVICES.map((s) => ({ value: s, label: serviceLabel(s) }))].map((t) => ({
            ...t,
            href: hrefWith("/logs", params, { service: t.value === "all" ? null : t.value, page: null }),
          }))}
        />
      }
    >
      <DataTable
        columns={COLUMNS}
        rows={rows}
        rowKey="id"
        toolbar={
          <ListFilters
            searchLabel="Search log messages"
            placeholder="Words in the message…"
            statusParam="level"
            statusLabel="Level"
            statuses={[
              { value: "ERROR", label: "Errors" },
              { value: "WARN", label: "Warnings" },
            ]}
          />
        }
        emptyMessage={filtered ? "No log line matches these filters." : "No warnings or errors logged. Services write them here as they happen."}
        serverPaginated
        currentPage={page - 1}
        hasMore={hasMore}
        prevHref={page > 1 ? hrefWith("/logs", params, { page: page - 1 }) : undefined}
        nextHref={hasMore ? hrefWith("/logs", params, { page: page + 1 }) : undefined}
      />
    </PageShell>
  );
}
