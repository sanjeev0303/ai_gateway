import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatCompact, formatNumber, formatPercent, formatUsd, microsToUsd } from "@/lib/format";
import { ROLE_ORDER, ROLE_TONE, STATUS_TONE, toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Users · Sentinel",
  description: "Who is calling models, and what it costs.",
};

interface UserRow {
  id: string;
  name: string;
  email: string;
  role: string;
  roleTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  requests: number;
  tokens: number;
  spend: number;
}

const userColumns: readonly Column<UserRow>[] = [
  { key: "name", header: "User", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role", tone: "roleTone", sort: "role" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "requests", header: "Requests 30d", primary: "requests", format: "number", align: "right", sort: "requests" },
  { key: "tokens", header: "Tokens 30d", primary: "tokens", format: "number", align: "right", sort: "tokens" },
  { key: "spend", header: "Spend 30d", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export default async function UsersPage() {
  const { members, usage } = await getApiClient();
  const [memberList, spendByUser] = await Promise.all([members.list(), usage.byUser()]);

  const totals = new Map(
    spendByUser.items.map((row) => [row.UserID, { requests: row.Requests, tokens: row.TotalTokens, spend: microsToUsd(row.CostMicros) }]),
  );

  const rows: UserRow[] = memberList.items.map((member) => {
    const totals_ = totals.get(member.user.id);
    const role = member.role;
    const status = member.user.status;
    return {
      id: member.user.id,
      name: member.user.display_name || member.user.email,
      email: member.user.email,
      role,
      roleTone: toneFor(ROLE_TONE, role),
      status,
      statusTone: toneFor(STATUS_TONE, status),
      requests: totals_?.requests ?? 0,
      tokens: totals_?.tokens ?? 0,
      spend: totals_?.spend ?? 0,
    };
  });

  const active = rows.filter((row) => row.status === "active").length;
  const totalSpend = rows.reduce((sum, row) => sum + row.spend, 0);
  const totalRequests = rows.reduce((sum, row) => sum + row.requests, 0);

  const spendByRole = ROLE_ORDER.map((role) => ({
    label: role,
    value: rows.filter((row) => row.role === role).reduce((sum, row) => sum + row.spend, 0),
  })).filter((datum) => datum.value > 0);

  const topSpenders = [...rows]
    .sort((a, b) => b.spend - a.spend)
    .slice(0, 6)
    .map((row) => ({ label: row.name, value: row.spend }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Users"
        description="Everyone with access to the gateway, and what they are spending."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active users"
            value={formatNumber(active)}
            hint={`${formatNumber(rows.length)} total accounts`}
          />
          <StatCard
            label="Requests"
            value={formatCompact(totalRequests)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint={`${formatUsd(totalSpend / Math.max(1, active))} per active user`}
          />
          <StatCard
            label="Avg spend"
            value={formatUsd(totalSpend / Math.max(1, rows.length))}
            tone="neutral"
            hint="Across all accounts"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Top spenders</CardTitle>
              <CardDescription>Trailing 30 days, highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={topSpenders} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by role</CardTitle>
              <CardDescription>
                Where spend concentrates by permission level.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={spendByRole}
                centerLabel={formatCompact(totalSpend)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>People</CardTitle>
            <CardDescription>
              Request and token counts are attributed per user over the trailing 30 days.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={userColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No users in this workspace."
            />
          </CardContent>
        </Card>

        <p className="text-xs text-muted-foreground">
          {formatPercent(active / Math.max(1, rows.length))} of accounts are active.
        </p>
      </PageShell>
    </div>
  );
}
