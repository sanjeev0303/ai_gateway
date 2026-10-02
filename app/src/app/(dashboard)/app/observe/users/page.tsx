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
import { formatCompact, formatNumber, formatPercent, formatRelative, formatUsd } from "@/lib/format";
import { NOW, teams, users } from "@/lib/mock-data";
import type { OrganizationRole, StatusTone, UserStatus } from "@/types";

export const metadata: Metadata = {
  title: "Users · Sentinel",
  description: "Who is calling models, on which teams, and what it costs.",
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

const teamNames = new Map(teams.map((team) => [team.id, team.name]));

interface UserRow {
  id: string;
  name: string;
  email: string;
  role: string;
  roleTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  teams: string;
  teamCount: number;
  projects: number;
  lastActive: string;
  lastActiveTs: number;
  spend: number;
  token: string;
}

const userColumns: readonly Column<UserRow>[] = [
  { key: "name", header: "User", primary: "name", secondary: "email", sort: "name" },
  { key: "role", header: "Role", primary: "role", tone: "roleTone", sort: "role" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "teams",
    header: "Teams",
    primary: "teams",
    sort: "teamCount",
  },
  {
    key: "projects",
    header: "Projects",
    primary: "projects",
    format: "number",
    align: "right",
    sort: "projects",
  },
  {
    key: "lastActive",
    header: "Last active",
    primary: "lastActive",
    sort: "lastActiveTs",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
  { key: "token", header: "Token", primary: "token", sort: "token" },
];

const rows: UserRow[] = users.map((user) => ({
  id: user.id,
  name: user.name,
  email: user.email,
  role: user.role,
  roleTone: ROLE_TONE[user.role],
  status: user.status,
  statusTone: USER_STATUS_TONE[user.status],
  teams:
    user.teamIds.length === 0
      ? "No team"
      : user.teamIds.map((id) => teamNames.get(id) ?? "Unknown").join(", "),
  teamCount: user.teamIds.length,
  projects: user.projectIds.length,
  lastActive: formatRelative(user.lastActiveAt, NOW),
  lastActiveTs: Date.parse(user.lastActiveAt),
  spend: user.spendMtd,
  token: user.hasPersonalToken ? "Personal" : "SSO only",
}));

const active = users.filter((user) => user.status === "active").length;
const invited = users.filter((user) => user.status === "invited").length;
const personalTokens = users.filter((user) => user.hasPersonalToken).length;
const totalSpend = users.reduce((acc, user) => acc + user.spendMtd, 0);

const spendByRole = Object.entries(
  users.reduce<Record<string, number>>((acc, user) => {
    acc[user.role] = (acc[user.role] ?? 0) + user.spendMtd;
    return acc;
  }, {}),
).map(([label, value]) => ({ label, value }));

const topSpenders = [...users]
  .sort((a, b) => b.spendMtd - a.spendMtd)
  .slice(0, 6)
  .map((user) => ({ label: user.name, value: user.spendMtd }));

export default function UsersPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Users"
        description="Everyone with access to the gateway, what they are spending, and whether they hold a personal token."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active users"
            value={formatNumber(active)}
            hint={`${formatNumber(users.length)} total accounts`}
          />
          <StatCard
            label="Pending invites"
            value={formatNumber(invited)}
            tone={invited > 0 ? "info" : "neutral"}
            hint="Have not signed in yet"
          />
          <StatCard
            label="Personal tokens"
            value={formatNumber(personalTokens)}
            tone={personalTokens > 0 ? "warning" : "success"}
            hint="Bypass SSO and rotate independently"
          />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint={`${formatUsd(totalSpend / Math.max(1, active))} per active user`}
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
              A personal token is a weaker control than SSO, so it is surfaced
              here rather than buried in settings.
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
          {formatPercent(active / Math.max(1, users.length))} of accounts are active.
        </p>
      </PageShell>
    </div>
  );
}
