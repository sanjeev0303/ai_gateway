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
import {
  formatCompact,
  formatNumber,
  formatPercent,
  formatUsd,
} from "@/lib/format";
import { models, projects, teams, users } from "@/lib/mock-data";
import type { ProjectEnvironment, StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Projects · Sentinel",
  description: "Every project routed through the gateway, with its budget and model allowlist.",
};

const ENVIRONMENT_TONE: Record<ProjectEnvironment, StatusTone> = {
  production: "success",
  staging: "info",
  development: "neutral",
};

const STATUS_TONE: Record<"active" | "archived", StatusTone> = {
  active: "success",
  archived: "neutral",
};

const teamName = new Map(teams.map((team) => [team.id, team.name]));
const userName = new Map(users.map((user) => [user.id, user.name]));

interface ProjectRow {
  id: string;
  name: string;
  slug: string;
  environment: string;
  environmentTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  team: string;
  owner: string;
  models: string;
  modelCount: number;
  spend: number;
  budget: string;
  utilization: string;
  usedFraction: number;
  meterTone: StatusTone;
}

const projectColumns: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name" },
  {
    key: "environment",
    header: "Environment",
    primary: "environment",
    tone: "environmentTone",
    sort: "environment",
  },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "team", header: "Team", primary: "team", sort: "team" },
  { key: "owner", header: "Owner", primary: "owner", sort: "owner" },
  {
    key: "models",
    header: "Models allowed",
    primary: "models",
    sort: "modelCount",
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
    align: "right",
    sort: "budget",
  },
  {
    key: "utilization",
    header: "Used",
    primary: "utilization",
    tone: "meterTone",
    meter: "usedFraction",
    meterMax: 1,
    align: "right",
    sort: "usedFraction",
  },
];

function meterTone(spend: number, budget: number | null): StatusTone {
  if (budget === null || budget <= 0) return "neutral";
  const ratio = spend / budget;
  if (ratio >= 1) return "danger";
  if (ratio >= 0.8) return "warning";
  return "success";
}

const rows: ProjectRow[] = projects.map((project) => {
  const ratio =
    project.monthlyBudget !== null && project.monthlyBudget > 0
      ? project.spendMtd / project.monthlyBudget
      : 0;
  return {
    id: project.id,
    name: project.name,
    slug: project.slug,
    environment: project.environment,
    environmentTone: ENVIRONMENT_TONE[project.environment],
    status: project.status,
    statusTone: STATUS_TONE[project.status],
    team: teamName.get(project.teamId) ?? "Unassigned",
    owner: userName.get(project.ownerId) ?? "Unassigned",
    models: formatNumber(project.allowedModelIds.length),
    modelCount: project.allowedModelIds.length,
    spend: project.spendMtd,
    budget:
      project.monthlyBudget === null ? "Unbudgeted" : formatUsd(project.monthlyBudget),
    utilization:
      project.monthlyBudget === null ? "—" : formatPercent(Math.min(1, ratio)),
    usedFraction: Math.min(1, ratio),
    meterTone: meterTone(project.spendMtd, project.monthlyBudget),
  };
});

const totalSpend = projects.reduce((acc, project) => acc + project.spendMtd, 0);
const active = projects.filter((project) => project.status === "active").length;
const production = projects.filter(
  (project) => project.environment === "production",
).length;
const overBudget = projects.filter(
  (project) =>
    project.monthlyBudget !== null && project.spendMtd >= project.monthlyBudget,
);

const spendByProject = [...projects]
  .sort((a, b) => b.spendMtd - a.spendMtd)
  .map((project) => ({ label: project.name, value: project.spendMtd }));

const spendByEnvironment = (["production", "staging", "development"] as const).map(
  (environment) => ({
    label: environment,
    value: projects
      .filter((project) => project.environment === environment)
      .reduce((acc, project) => acc + project.spendMtd, 0),
  }),
);

/** Models allowed anywhere but not granted to every project. */
const broadlyAllowed = new Set(
  projects.flatMap((project) => project.allowedModelIds),
);
const catalog = models.filter((model) => broadlyAllowed.has(model.id));

export default function ProjectsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Every project routed through the gateway, with the models it may call and the budget it is held to."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active projects"
            value={formatNumber(active)}
            hint={`${formatNumber(projects.length)} total`}
          />
          <StatCard
            label="In production"
            value={formatNumber(production)}
            tone={production > 0 ? "success" : "neutral"}
            hint="Serving live traffic"
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
            hint={`${formatUsd(totalSpend / Math.max(1, active))} per active project`}
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Trailing 30 days, highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendByProject} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by environment</CardTitle>
              <CardDescription>
                Staging and development should be a small share of the bill.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={spendByEnvironment}
                centerLabel={formatCompact(totalSpend)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Projects</CardTitle>
            <CardDescription>
              A project can only call the models on its allowlist, regardless of
              what the workspace allows.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={projectColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No projects configured."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Model allowlist coverage</CardTitle>
            <CardDescription>
              {formatNumber(catalog.length)} of {formatNumber(models.length)}{" "}
              catalogued models are allowed by at least one project. A model
              nobody can reach is dead weight; one every project can reach is
              effectively unrestricted.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart
              data={models
                .filter((model) => model.callCount30d > 0)
                .sort((a, b) => b.callCount30d - a.callCount30d)
                .slice(0, 8)
                .map((model) => ({
                  label: model.displayName,
                  value: projects.filter((project) =>
                    project.allowedModelIds.includes(model.id),
                  ).length,
                }))}
              formatValue={(value) => `${formatNumber(value)} projects`}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
