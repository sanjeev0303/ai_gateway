import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import { CreateTeamDialog } from "./teams-client";

export const metadata: Metadata = {
  title: "Teams · Sentinel",
  description: "Groups that own workloads and accrue spend.",
};

interface TeamRow {
  id: string;
  name: string;
  description: string;
  members: number;
  requests: number;
  tokens: number;
  spend: number;
  detailUrl: string;
}

const COLUMNS: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "description", sort: "name", href: "detailUrl" },
  { key: "members", header: "Members", primary: "members", format: "number", align: "right", sort: "members" },
  { key: "requests", header: "Requests", primary: "requests", format: "compact", align: "right", sort: "requests" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "spend", header: "Spend (30d)", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export default async function TeamsPage() {
  const { teams, usage } = await getApiClient();
  const [teamList, teamUsage] = await Promise.all([teams.list(), usage.byTeam()]);

  const spendByTeam = new Map(teamUsage.items.map((row) => [row.TeamID, row]));
  const memberCounts = new Map(
    await Promise.all(
      teamList.items.map(
        async (team) => [team.id, (await teams.listMembers(team.id)).items.length] as const,
      ),
    ),
  );

const rows: TeamRow[] = teamList.items.map((team) => {
    const attributed = spendByTeam.get(team.id);
    return {
      id: team.id,
      name: team.name,
      description: team.description,
      members: memberCounts.get(team.id) ?? 0,
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      spend: microsToUsd(attributed?.CostMicros ?? 0),
      detailUrl: `/app/organization/teams/${team.id}`,
    };
  });

  const totalMembers = rows.reduce((sum, row) => sum + row.members, 0);
  const totalRequests = rows.reduce((sum, row) => sum + row.requests, 0);
  const totalSpend = rows.reduce((sum, row) => sum + row.spend, 0);
  const topSpend = [...rows].sort((a, b) => b.spend - a.spend).slice(0, 8);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Groups that own projects and are the unit of accountability for spend."
        actions={
          <CreateTeamDialog>
            <Button size="sm">
              <Plus aria-hidden /> New team
            </Button>
          </CreateTeamDialog>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Teams" value={String(rows.length)} hint="Across the organization" />
          <StatCard
            label="Seats filled"
            value={formatNumber(totalMembers)}
            hint="Team memberships held"
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Spend"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>Trailing 30-day spend, top eight teams.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={topSpend.map((row) => ({ label: row.name, value: row.spend }))}
                formatValue={(value) => formatUsd(value)}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Membership</CardTitle>
              <CardDescription>People per team.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={rows.map((row) => ({ label: row.name, value: row.members }))}
                centerLabel="members"
                size={190}
              />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>All teams</CardTitle>
            <CardDescription>
              {formatNumber(totalMembers)} people are distributed across these groups.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              emptyMessage="No teams created yet."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
