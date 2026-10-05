import type { Metadata } from "next";

import { BarChart } from "@/components/charts";
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
import { formatNumber, formatPercent, formatUsd, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";

export const metadata: Metadata = {
  title: "Teams · Sentinel",
  description: "The spend each team owns, and how the bill divides between them.",
};

interface TeamRow {
  id: string;
  name: string;
  description: string;
  members: number;
  requests: number;
  tokens: number;
  share: string;
  shareFraction: number;
  spend: number;
}

const teamColumns: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "description", sort: "name" },
  {
    key: "members",
    header: "Members",
    primary: "members",
    format: "number",
    sort: "members",
    align: "right",
  },
  {
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "compact",
    sort: "requests",
    align: "right",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    sort: "tokens",
    align: "right",
  },
  {
    key: "share",
    header: "Share",
    primary: "share",
    meter: "shareFraction",
    meterMax: 1,
    sort: "shareFraction",
    align: "right",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
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

  const totalSpend = teamUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );
  const totalRequests = teamUsage.items.reduce((sum, row) => sum + row.Requests, 0);
  const totalTokens = teamUsage.items.reduce((sum, row) => sum + row.TotalTokens, 0);

  const rows: TeamRow[] = teamList.items.map((team) => {
    const attributed = spendByTeam.get(team.id);
    const spend = microsToUsd(attributed?.CostMicros ?? 0);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: team.id,
      name: team.name,
      description: team.description,
      members: memberCounts.get(team.id) ?? 0,
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      share: formatPercent(shareFraction),
      shareFraction,
      spend,
    };
  });

  const spendByTeamChart = [...rows]
    .sort((a, b) => b.spend - a.spend)
    .map((team) => ({ label: team.name, value: team.spend }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Who owns which spend. One team holding most of the bill is the thing to look at."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Teams"
            value={formatNumber(rows.length)}
            hint="With attributed usage"
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Trailing 30 days"
          />
          <StatCard label="Tokens" value={formatNumber(totalTokens)} hint="Trailing 30 days" />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint="Across every team"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Spend by team</CardTitle>
            <CardDescription>Trailing 30 days, highest first.</CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart data={spendByTeamChart} formatValue={formatUsd} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Teams</CardTitle>
            <CardDescription>
              Each team&rsquo;s share of organization spend over the trailing 30 days.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={teamColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No teams configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
