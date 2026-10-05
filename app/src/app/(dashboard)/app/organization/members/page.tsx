import type { Metadata } from "next";
import { UserPlus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { ROLE_ORDER, ROLE_TONE, STATUS_TONE, toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Members · Sentinel",
  description: "Everyone in the organization, with their role, status, and attributed spend.",
};

interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  roleTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  spend: number;
}

const COLUMNS: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role", tone: "roleTone" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "spend", header: "Spend (30d)", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export default async function MembersPage() {
  const { members, usage } = await getApiClient();
  const [memberList, spendByUser] = await Promise.all([members.list(), usage.byUser()]);

  const spend = new Map(
    spendByUser.items.map((row) => [row.UserID, microsToUsd(row.CostMicros)]),
  );

  const rows: MemberRow[] = memberList.items.map((member) => ({
    id: member.user.id,
    name: member.user.display_name || member.user.email,
    email: member.user.email,
    role: member.role,
    roleTone: toneFor(ROLE_TONE, member.role),
    status: member.user.status,
    statusTone: toneFor(STATUS_TONE, member.user.status),
    spend: spend.get(member.user.id) ?? 0,
  }));

  const active = rows.filter((row) => row.status === "active");
  const admins = rows.filter((row) => row.role === "owner" || row.role === "admin");
  const totalSpend = rows.reduce((sum, row) => sum + row.spend, 0);

  const byRole = ROLE_ORDER.map((role) => ({
    label: role,
    value: rows.filter((row) => row.role === role).length,
  })).filter((datum) => datum.value > 0);

  const topSpenders = [...rows]
    .sort((a, b) => b.spend - a.spend)
    .slice(0, 6)
    .map((row) => ({ label: row.name, value: row.spend }));

  const disabled = rows.filter((row) => row.status === "disabled");

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Members"
        description="Every seat in the organization, and what each person is accountable for."
        actions={
          <Button size="sm">
            <UserPlus aria-hidden /> Invite member
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active members"
            value={formatNumber(active.length)}
            tone="success"
            hint="Enabled and signing in"
          />
          <StatCard label="Admins" value={String(admins.length)} hint="Owners and admins" />
          <StatCard
            label="Attributed spend"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Headcount by role</CardTitle>
              <CardDescription>How access is distributed across the organization.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart data={byRole} centerLabel="members" size={190} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Top spenders</CardTitle>
              <CardDescription>Highest attributed model spend, trailing 30 days.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={topSpenders} formatValue={(value) => formatUsd(value)} />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>All members</CardTitle>
            <CardDescription>
              Removing a member revokes their tokens but keeps their audit history intact.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              toolbar={
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge tone="success">{active.length} active</StatusBadge>
                  {disabled.length > 0 && (
                    <StatusBadge tone="warning">{disabled.length} disabled</StatusBadge>
                  )}
                </div>
              }
              emptyMessage="No members in this organization."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
