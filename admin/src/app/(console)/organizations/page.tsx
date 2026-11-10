import type { Metadata } from "next";

import type { AdminOrganization, AdminPage } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { RangeTabs } from "@ai-governance/console-ui/dashboard/range-tabs";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { ListFilters } from "../_components/list-filters";
import { hrefWith, PAGE_SIZE, pageFrom, param, usd, WINDOW_LABEL, windowFrom, type SearchParams } from "../_lib";

export const metadata: Metadata = { title: "Organizations · Platform console" };

interface OrgRow {
  id: string;
  name: string;
  slug: string;
  href: string;
  status: string;
  statusTone: StatusTone;
  plan: string;
  members: number;
  keys: number;
  requests: number;
  spend: string;
  createdAt: string;
  lastRequest: string;
  lastRequestAt: string;
}

const COLUMNS: Column<OrgRow>[] = [
  { key: "name", header: "Organization", primary: "name", secondary: "slug", href: "href", grow: true },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "plan", header: "Plan", primary: "plan" },
  { key: "members", header: "Members", primary: "members", format: "number", align: "right" },
  { key: "keys", header: "Active keys", primary: "keys", format: "number", align: "right" },
  { key: "requests", header: "Requests", primary: "requests", format: "compact", align: "right" },
  { key: "spend", header: "Spend", primary: "spend", align: "right" },
  { key: "last", header: "Last request", primary: "lastRequest", sort: "lastRequestAt" },
  { key: "created", header: "Created", primary: "createdAt", format: "date" },
];

function toRow(org: AdminOrganization): OrgRow {
  return {
    id: org.id,
    name: org.name,
    slug: org.slug,
    href: `/organizations/${org.id}`,
    status: org.status === "suspended" ? "Suspended" : "Active",
    statusTone: org.status === "suspended" ? "danger" : "success",
    plan: org.plan_name || "No plan",
    members: org.members,
    keys: org.active_keys,
    requests: org.requests,
    spend: usd(org.cost_micros),
    createdAt: org.created_at,
    lastRequest: org.last_request_at ? formatRelative(org.last_request_at) : "No traffic",
    lastRequestAt: org.last_request_at ?? "",
  };
}

export default async function AdminOrganizationsPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const window = windowFrom(params);
  const { page, offset } = pageFrom(params);
  const search = param(params, "search");
  const status = param(params, "status");

  let result: AdminPage<AdminOrganization>;
  try {
    const { admin } = await getApiClient();
    result = await admin.organizations({ search, status, window, limit: PAGE_SIZE, offset });
  } catch {
    return (
      <PageShell title="Organizations">
        <ErrorState description="The organization list could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const filtered = search !== "" || status !== "";
  const hasMore = offset + result.items.length < result.total;

  return (
    <PageShell
      title="Organizations"
      description={`${formatNumber(result.total)} ${filtered ? "matching" : "in total"}. Requests and spend cover the ${WINDOW_LABEL[window]}.`}
      actions={<RangeTabs />}
    >
      <DataTable
        columns={COLUMNS}
        rows={result.items.map(toRow)}
        rowKey="id"
        toolbar={
          <ListFilters
            searchLabel="Search organizations"
            placeholder="Name, slug or ID…"
            statuses={[
              { value: "active", label: "Active" },
              { value: "suspended", label: "Suspended" },
            ]}
          />
        }
        emptyMessage={filtered ? "No organization matches these filters." : "No organization has signed up yet."}
        serverPaginated
        currentPage={page - 1}
        hasMore={hasMore}
        prevHref={page > 1 ? hrefWith("/organizations", params, { page: page - 1 }) : undefined}
        nextHref={hasMore ? hrefWith("/organizations", params, { page: page + 1 }) : undefined}
      />
    </PageShell>
  );
}
