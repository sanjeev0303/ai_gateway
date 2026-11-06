import type { Metadata } from "next";

import {
  DataTable,
  ObserveFilters,
  PageShell,
  StatCard,
  type Column,
  BreakdownTabs,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatPercent, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";

export const metadata: Metadata = {
  title: "Teams · Tollseal",
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
  detailUrl: string;
}

const teamColumns: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "description", sort: "name", href: "detailUrl" },
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

export default async function TeamsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    service_account_id: typeof params.service_account_id === "string" ? params.service_account_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

  const { teams, usage } = await getApiClient();
  const [teamList, teamUsage] = await Promise.all([teams.list(), usage.byTeam(filters)]);

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
      detailUrl: `/app/organization/teams/${team.id}`,
    };
  });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Who owns which spend. One team holding most of the bill is the thing to look at."
      >
        <BreakdownTabs />
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard
            label="Teams"
            value={formatNumber(rows.length)}
            hint="With attributed usage"
          />
        </section>

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
