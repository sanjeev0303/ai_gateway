import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
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
import { Progress } from "@/components/ui/progress";
import { formatNumber, formatPercent, formatUsd } from "@/lib/format";
import { teams, users } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Teams · Sentinel",
  description: "Groups that own workloads, carry budgets, and accrue spend.",
};

const userName = new Map(users.map((user) => [user.id, user.name]));

interface TeamRow {
  id: string;
  name: string;
  slug: string;
  lead: string;
  members: number;
  projects: number;
  budget: string;
  spend: number;
  burnPct: number;
  budgetTone: StatusTone;
}

const COLUMNS: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "slug", sort: "name" },
  { key: "lead", header: "Lead", primary: "lead" },
  { key: "members", header: "Members", primary: "members", align: "right" },
  { key: "projects", header: "Projects", primary: "projects", align: "right" },
  { key: "budget", header: "Budget", primary: "budget", align: "right" },
  { key: "spend", header: "Spend MTD", primary: "spend", format: "usd", align: "right", sort: "spend" },
  { key: "burnPct", header: "Budget used", primary: "burnPct", format: "percentRaw", align: "right", meter: "burnPct", meterMax: 100, sort: "burnPct" },
];

export default function TeamsPage() {
  const rows: TeamRow[] = teams.map((team) => {
    const burnPct = team.monthlyBudget
      ? (team.spendMtd / team.monthlyBudget) * 100
      : 0;
    return {
      id: team.id,
      name: team.name,
      slug: team.slug,
      lead: userName.get(team.leadId) ?? "Unassigned",
      members: team.memberIds.length,
      projects: team.projectIds.length,
      budget: team.monthlyBudget === null ? "Unbudgeted" : formatUsd(team.monthlyBudget),
      spend: team.spendMtd,
      burnPct: Math.round(burnPct),
      budgetTone: burnPct >= 95 ? "danger" : burnPct >= 80 ? "warning" : "success",
    };
  });

  const totalBudget = teams.reduce((sum, team) => sum + (team.monthlyBudget ?? 0), 0);
  const totalSpend = teams.reduce((sum, team) => sum + team.spendMtd, 0);
  const overBudget = rows.filter((row) => row.burnPct >= 80);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Groups that own projects, hold budget, and are the unit of accountability for spend."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> New team
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Teams" value={String(teams.length)} hint="Across the organization" />
          <StatCard
            label="Total budget"
            value={formatUsd(totalBudget)}
            hint="Monthly, budgeted teams only"
          />
          <StatCard
            label="Spend MTD"
            value={formatUsd(totalSpend)}
            hint={formatPercent((totalSpend / totalBudget) * 100, 0) + " of allocated budget"}
          />
          <StatCard
            label="Near budget"
            value={String(overBudget.length)}
            tone={overBudget.length > 0 ? "warning" : "success"}
            hint="At or above 80% burn"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>Trailing 30-day spend against monthly budget.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={[...teams]
                  .sort((a, b) => b.spendMtd - a.spendMtd)
                  .map((team) => ({
                    label: team.name,
                    value: team.spendMtd,
                    secondary: team.monthlyBudget ?? 0,
                  }))}
                showSecondary
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
                data={teams.map((team) => ({
                  label: team.name,
                  value: team.memberIds.length,
                }))}
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
              {formatNumber(users.length)} people are distributed across these groups.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <p className="text-sm text-muted-foreground">
                  {overBudget.length} team{overBudget.length === 1 ? "" : "s"} at or above 80% of budget.
                </p>
              }
              emptyMessage="No teams created yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Budget burn</CardTitle>
            <CardDescription>Month-to-date consumption of each team allocation.</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {rows.map((row) => (
                <li key={row.id} className="space-y-1.5">
                  <div className="flex items-center justify-between gap-3 text-sm">
                    <span className="truncate font-medium">{row.name}</span>
                    <span className="tabular-nums text-muted-foreground">
                      {formatUsd(row.spend)} of {row.budget}
                    </span>
                  </div>
                  <Progress
                    value={Math.min(row.burnPct, 100)}
                    className={row.budgetTone === "danger" ? "[&>div]:bg-rose-500" : row.budgetTone === "warning" ? "[&>div]:bg-amber-500" : undefined}
                  />
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
