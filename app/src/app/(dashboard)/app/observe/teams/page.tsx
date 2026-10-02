import type { Metadata } from "next";

import { BarChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import { Progress } from "@/components/ui/progress";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatPercent, formatUsd } from "@/lib/format";
import { teams, users } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Teams · Sentinel",
  description: "Team membership, budgets, and the spend each team owns.",
};

function utilization(spend: number, budget: number | null): number {
  if (budget === null || budget <= 0) return 0;
  return spend / budget;
}

function budgetTone(spend: number, budget: number | null): StatusTone {
  if (budget === null) return "neutral";
  const ratio = utilization(spend, budget);
  if (ratio >= 1) return "danger";
  if (ratio >= 0.8) return "warning";
  return "success";
}

const userName = new Map(users.map((user) => [user.id, user.name]));

interface TeamRow {
  id: string;
  name: string;
  slug: string;
  lead: string;
  members: string;
  memberCount: number;
  projects: string;
  projectCount: number;
  spend: number;
  budget: string;
  budgetTone: StatusTone;
  used: string;
  usedFraction: number;
}

const teamColumns: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "slug", sort: "name" },
  { key: "lead", header: "Lead", primary: "lead", sort: "lead" },
  {
    key: "members",
    header: "Members",
    primary: "members",
    sort: "memberCount",
  },
  {
    key: "projects",
    header: "Projects",
    primary: "projects",
    sort: "projectCount",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
  {
    key: "budget",
    header: "Budget",
    primary: "budget",
    tone: "budgetTone",
    align: "right",
    sort: "budget",
  },
  {
    key: "used",
    header: "Used",
    primary: "used",
    meter: "usedFraction",
    meterMax: 1,
    align: "right",
    sort: "usedFraction",
  },
];

const rows: TeamRow[] = teams.map((team) => {
  const ratio = utilization(team.spendMtd, team.monthlyBudget);
  return {
    id: team.id,
    name: team.name,
    slug: team.slug,
    lead: userName.get(team.leadId) ?? "Unassigned",
    members: formatNumber(team.memberIds.length),
    memberCount: team.memberIds.length,
    projects: formatNumber(team.projectIds.length),
    projectCount: team.projectIds.length,
    spend: team.spendMtd,
    budget: team.monthlyBudget === null ? "Unbudgeted" : formatUsd(team.monthlyBudget),
    budgetTone: budgetTone(team.spendMtd, team.monthlyBudget),
    used: team.monthlyBudget === null ? "—" : formatPercent(ratio),
    usedFraction: ratio,
  };
});

const spendByTeam = [...teams]
  .sort((a, b) => b.spendMtd - a.spendMtd)
  .map((team) => ({ label: team.name, value: team.spendMtd }));

const totalSpend = teams.reduce((acc, team) => acc + team.spendMtd, 0);
const totalMembers = new Set(teams.flatMap((team) => team.memberIds)).size;
const budgeted = teams.filter((team) => team.monthlyBudget !== null);
const overBudget = budgeted.filter(
  (team) => team.monthlyBudget !== null && team.spendMtd >= team.monthlyBudget,
);

export default function TeamsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Who owns which spend. A team without a budget is a team nobody is watching."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Teams"
            value={formatNumber(teams.length)}
            hint={`${formatNumber(totalMembers)} distinct members`}
          />
          <StatCard
            label="Budgeted teams"
            value={`${formatNumber(budgeted.length)} of ${formatNumber(teams.length)}`}
            tone={budgeted.length === teams.length ? "success" : "warning"}
            hint="Have a monthly cap"
          />
          <StatCard
            label="Over budget"
            value={formatNumber(overBudget.length)}
            tone={overBudget.length > 0 ? "danger" : "success"}
            hint="Spend at or above cap"
          />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint="Across every team"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>Trailing 30 days, highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendByTeam} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Budget headroom</CardTitle>
              <CardDescription>Consumed share of each monthly cap.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {budgeted.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No team has a budget cap configured.
                </p>
              ) : (
                budgeted
                  .slice()
                  .sort(
                    (a, b) =>
                      utilization(b.spendMtd, b.monthlyBudget) -
                      utilization(a.spendMtd, a.monthlyBudget),
                  )
                  .map((team) => {
                    const ratio = Math.min(1, utilization(team.spendMtd, team.monthlyBudget));
                    return (
                      <div key={team.id} className="space-y-1.5">
                        <div className="flex items-center justify-between text-sm">
                          <span className="truncate">{team.name}</span>
                          <span className="tabular-nums text-muted-foreground">
                            {formatPercent(utilization(team.spendMtd, team.monthlyBudget))}
                          </span>
                        </div>
                        <Progress value={ratio * 100} />
                      </div>
                    );
                  })
              )}
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Teams</CardTitle>
            <CardDescription>
              Spend, membership, and the cap each team is held to.
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

        <p className="text-xs text-muted-foreground">
          {formatNumber(teams.filter((team) => team.monthlyBudget === null).length)} unbudgeted
          teams account for{" "}
          {formatUsd(
            teams
              .filter((team) => team.monthlyBudget === null)
              .reduce((acc, team) => acc + team.spendMtd, 0),
          )}{" "}
          of spend with no cap to alert on.
        </p>
      </PageShell>
    </div>
  );
}
