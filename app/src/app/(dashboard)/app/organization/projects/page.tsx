import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatUsd } from "@/lib/format";
import { models, projects, teams, users } from "@/lib/mock-data";
import type { ProjectEnvironment, StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Projects · Sentinel",
  description: "Workloads that map spend, policy, and model access onto a real system of record.",
};

const teamName = new Map(teams.map((team) => [team.id, team.name]));
const ownerName = new Map(users.map((user) => [user.id, user.name]));
const modelName = new Map(models.map((model) => [model.id, model.displayName]));

const ENV_TONE: Record<ProjectEnvironment, StatusTone> = {
  production: "success",
  staging: "info",
  development: "neutral",
};

interface ProjectRow {
  id: string;
  name: string;
  slug: string;
  description: string;
  environment: string;
  envTone: StatusTone;
  team: string;
  owner: string;
  models: string;
  modelCount: number;
  status: string;
  statusTone: StatusTone;
  spend: number;
  budget: string;
  burnPct: number;
  budgetTone: StatusTone;
}

const COLUMNS: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name" },
  { key: "environment", header: "Environment", primary: "environment", tone: "envTone" },
  { key: "team", header: "Team", primary: "team" },
  { key: "owner", header: "Owner", primary: "owner" },
  { key: "modelCount", header: "Models", primary: "modelCount", align: "right" },
  { key: "spend", header: "Spend MTD", primary: "spend", format: "usd", align: "right", sort: "spend" },
  { key: "budget", header: "Budget", primary: "budget", align: "right" },
  { key: "burnPct", header: "Budget used", primary: "burnPct", format: "percentRaw", align: "right", meter: "burnPct", meterMax: 100, sort: "burnPct" },
];

export default function ProjectsPage() {
  const rows: ProjectRow[] = projects.map((project) => {
    const burnPct = project.monthlyBudget
      ? (project.spendMtd / project.monthlyBudget) * 100
      : 0;
    return {
      id: project.id,
      name: project.name,
      slug: project.slug,
      description: project.description,
      environment: project.environment,
      envTone: ENV_TONE[project.environment],
      team: teamName.get(project.teamId) ?? "Unassigned",
      owner: ownerName.get(project.ownerId) ?? "Unassigned",
      models: project.allowedModelIds.map((id) => modelName.get(id) ?? id).join(", "),
      modelCount: project.allowedModelIds.length,
      status: project.status,
      statusTone: project.status === "active" ? "success" : "neutral",
      spend: project.spendMtd,
      budget: project.monthlyBudget === null ? "Unbudgeted" : formatUsd(project.monthlyBudget),
      burnPct: Math.round(burnPct),
      budgetTone: burnPct >= 95 ? "danger" : burnPct >= 80 ? "warning" : "success",
    };
  });

  const active = projects.filter((project) => project.status === "active");
  const production = projects.filter((project) => project.environment === "production");
  const totalSpend = projects.reduce((sum, project) => sum + project.spendMtd, 0);
  const overBudget = rows.filter((row) => row.burnPct >= 80);

  const byEnvironment = (["production", "staging", "development"] as const)
    .map((environment) => ({
      label: environment,
      value: projects.filter((project) => project.environment === environment).length,
    }))
    .filter((datum) => datum.value > 0);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Each project is a budget holder, a policy scope, and an explicit allow-list of models."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> New project
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Projects"
            value={String(projects.length)}
            hint={`${active.length} active`}
          />
          <StatCard
            label="Production"
            value={String(production.length)}
            tone="success"
            hint="Serving live traffic"
          />
          <StatCard
            label="Spend MTD"
            value={formatUsd(totalSpend)}
            hint="Across every environment"
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
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Top eight projects by trailing 30-day spend.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={[...projects]
                  .sort((a, b) => b.spendMtd - a.spendMtd)
                  .slice(0, 8)
                  .map((project) => ({ label: project.name, value: project.spendMtd }))}
                formatValue={(value) => formatUsd(value)}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>By environment</CardTitle>
              <CardDescription>How projects are distributed across environments.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart data={byEnvironment} centerLabel="projects" size={190} />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>All projects</CardTitle>
            <CardDescription>
              Model access is an allow-list; anything not listed is denied at the gateway.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No projects created yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Model access by project</CardTitle>
            <CardDescription>The explicit allow-list enforced for each workload.</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {rows.slice(0, 8).map((row) => (
                <li key={row.id} className="flex flex-wrap items-center gap-2 border-b pb-3 last:border-0 last:pb-0">
                  <StatusBadge tone={row.envTone}>{row.environment}</StatusBadge>
                  <span className="text-sm font-medium">{row.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {row.team} · {row.owner}
                  </span>
                  <span className="ml-auto flex flex-wrap gap-1.5">
                    {row.models ? (
                      row.models.split(", ").map((model) => (
                        <Badge key={model} variant="outline" className="font-normal">
                          {model}
                        </Badge>
                      ))
                    ) : (
                      <Badge variant="outline" className="font-normal text-muted-foreground">
                        No models permitted
                      </Badge>
                    )}
                  </span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
