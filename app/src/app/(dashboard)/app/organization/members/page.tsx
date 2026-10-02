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
import { formatNumber, formatPercent, formatUsd } from "@/lib/format";
import { organization, teams, users } from "@/lib/mock-data";
import type { OrganizationRole, StatusTone, UserStatus } from "@/types";

export const metadata: Metadata = {
  title: "Members · Sentinel",
  description: "Everyone in the organization, with their role, status, and attributed spend.",
};

const USER_STATUS_TONE: Record<UserStatus, StatusTone> = {
  active: "success",
  invited: "info",
  suspended: "warning",
  deactivated: "neutral",
};

const ROLE_TONE: Record<OrganizationRole, StatusTone> = {
  owner: "info",
  admin: "info",
  member: "neutral",
  viewer: "neutral",
};

const teamName = new Map(teams.map((team) => [team.id, team.name]));

interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  roleTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  teams: string;
  projects: number;
  spend: number;
}

const COLUMNS: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role", tone: "roleTone" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "teams", header: "Teams", primary: "teams" },
  { key: "projects", header: "Projects", primary: "projects", align: "right" },
  { key: "spend", header: "Spend MTD", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export default function MembersPage() {
  const rows: MemberRow[] = users.map((user) => ({
    id: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
    roleTone: ROLE_TONE[user.role],
    status: user.status,
    statusTone: USER_STATUS_TONE[user.status],
    teams:
      user.teamIds.length > 0
        ? user.teamIds.map((id) => teamName.get(id) ?? id).join(", ")
        : "No team",
    projects: user.projectIds.length,
    spend: user.spendMtd,
  }));

  const active = users.filter((user) => user.status === "active");
  const admins = users.filter((user) => user.role === "owner" || user.role === "admin");
  const seatPct = (organization.seatsUsed / organization.seatsTotal) * 100;

  const byRole = (Object.keys(ROLE_TONE) as OrganizationRole[])
    .map((role) => ({ label: role, value: users.filter((user) => user.role === role).length }))
    .filter((datum) => datum.value > 0);

  const topSpenders = [...users]
    .sort((a, b) => b.spendMtd - a.spendMtd)
    .slice(0, 6)
    .map((user) => ({ label: user.name, value: user.spendMtd }));

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
            label="Seats in use"
            value={`${organization.seatsUsed} / ${organization.seatsTotal}`}
            hint={formatPercent(seatPct, 0) + " of the plan allocation"}
          />
          <StatCard label="Active members" value={formatNumber(active.length)} tone="success" hint="Enabled and signing in" />
          <StatCard label="Admins" value={String(admins.length)} hint="Owners and admins" />
          <StatCard
            label="Attributed spend"
            value={formatUsd(users.reduce((sum, user) => sum + user.spendMtd, 0))}
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
              <CardDescription>Highest attributed model spend this month.</CardDescription>
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
                  {users.some((user) => user.status === "invited") && (
                    <StatusBadge tone="info">
                      {users.filter((user) => user.status === "invited").length} invited
                    </StatusBadge>
                  )}
                  {users.some((user) => user.status === "suspended") && (
                    <StatusBadge tone="warning">
                      {users.filter((user) => user.status === "suspended").length} suspended
                    </StatusBadge>
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
