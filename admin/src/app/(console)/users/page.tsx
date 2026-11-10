import type { Metadata } from "next";

import type { AdminPage, AdminUser } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient, getPlatformAdmin } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { ListFilters } from "../_components/list-filters";
import { hrefWith, PAGE_SIZE, pageFrom, param, type SearchParams } from "../_lib";
import { UserActionsCell, type UserActionRow } from "./user-actions";

export const metadata: Metadata = { title: "Users · Platform console" };

interface UserRow extends UserActionRow {
  email: string;
  status: string;
  statusTone: StatusTone;
  orgs: string;
  orgCount: number;
  lastLogin: string;
  lastLoginAt: string;
  createdAt: string;
}

const COLUMNS: Column<UserRow>[] = [
  { key: "name", header: "User", primary: "name", secondary: "email", grow: true },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "orgs", header: "Organizations", primary: "orgCount", secondary: "orgs", format: "number" },
  { key: "login", header: "Last sign-in", primary: "lastLogin", sort: "lastLoginAt" },
  { key: "created", header: "Joined", primary: "createdAt", format: "date" },
];

export default async function AdminUsersPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const { page, offset } = pageFrom(params);
  const search = param(params, "search");
  const status = param(params, "status");

  let result: AdminPage<AdminUser>;
  let selfId = "";
  try {
    const { admin } = await getApiClient();
    const [list, me] = await Promise.all([admin.users({ search, status, limit: PAGE_SIZE, offset }), getPlatformAdmin()]);
    result = list;
    selfId = me?.id ?? "";
  } catch {
    return (
      <PageShell title="Users">
        <ErrorState description="The user list could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const rows: UserRow[] = result.items.map((u) => ({
    id: u.id,
    name: u.display_name || u.email,
    email: u.email,
    disabled: u.status === "disabled",
    isSelf: u.id === selfId,
    status: u.status === "disabled" ? "Disabled" : "Active",
    statusTone: u.status === "disabled" ? "danger" : "success",
    orgs: u.org_names || "No organization",
    orgCount: u.org_count,
    lastLogin: u.last_login_at ? formatRelative(u.last_login_at) : "Never",
    lastLoginAt: u.last_login_at ?? "",
    createdAt: u.created_at,
  }));

  const filtered = search !== "" || status !== "";
  const hasMore = offset + result.items.length < result.total;

  return (
    <PageShell
      title="Users"
      description={`${formatNumber(result.total)} ${filtered ? "matching" : "people on the platform"}. Disabling someone signs them out of every organization.`}
    >
      <DataTable
        columns={COLUMNS}
        rows={rows}
        rowKey="id"
        ActionsComponent={UserActionsCell}
        toolbar={
          <ListFilters
            searchLabel="Search users"
            placeholder="Email, name or ID…"
            statuses={[
              { value: "active", label: "Active" },
              { value: "disabled", label: "Disabled" },
            ]}
          />
        }
        emptyMessage={filtered ? "No user matches these filters." : "No one has signed up yet."}
        serverPaginated
        currentPage={page - 1}
        hasMore={hasMore}
        prevHref={page > 1 ? hrefWith("/users", params, { page: page - 1 }) : undefined}
        nextHref={hasMore ? hrefWith("/users", params, { page: page + 1 }) : undefined}
      />
    </PageShell>
  );
}
