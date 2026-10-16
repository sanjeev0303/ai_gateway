import { Metadata } from "next";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatNumber, formatMs, formatPercent, formatDate, formatRelative, microsToUsd } from "@/lib/format";
import type { StatusTone } from "@/types";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface UsageBreakdownRow {
  id: string;
  name: string;
  href: string;
  calls: number;
  tokens: number;
  cost: number;
}

const breakdownColumns: readonly Column<UsageBreakdownRow>[] = [
  { key: "name", header: "Name", primary: "name", sort: "name", href: "href" },
  { key: "calls", header: "Requests", primary: "calls", format: "compact", align: "right", sort: "calls" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "cost", header: "Spend", primary: "cost", format: "usd", align: "right", sort: "cost" },
];

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  const decoded = decodeURIComponent(id);
  return { title: `Model ${decoded} · Sentinel` };
}

export default async function ModelDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const modelSlug = decodeURIComponent(id);
  const { usage, models, projects, teams, members } = await getApiClient();

  try {
    const [
      summary,
      latencyData,
      byProject,
      byTeam,
      byUser,
      catalogList,
      projectList,
      teamList,
      memberList,
    ] = await Promise.all([
      usage.summary({ model: modelSlug }),
      usage.latency({ model: modelSlug }),
      usage.byProject({ model: modelSlug }),
      usage.byTeam({ model: modelSlug }),
      usage.byUser({ model: modelSlug }),
      models.list(),
      projects.list(),
      teams.list(),
      members.list(),
    ]);

    const catalogEntry = catalogList.items.find(
      (m) => m.model_slug === modelSlug || m.id === modelSlug
    );
    const displayName = catalogEntry?.display_name ?? modelSlug;

    const totalRequests = summary?.Requests ?? 0;
    const totalTokens = summary?.TotalTokens ?? 0;
    const totalCost = microsToUsd(summary?.TotalCostMicros ?? 0);
    const errorCount = summary?.Errors ?? 0;
    const errorRate = totalRequests > 0 ? errorCount / totalRequests : 0;

    const latencyOverall = latencyData?.items?.find((r) => !r.Provider && !r.Model) ?? latencyData?.items?.[0];
    const p95Latency = latencyOverall?.P95TotalMs ?? latencyOverall?.AvgTotalMs;

    const projectLookup = new Map(projectList.items.map((p) => [p.id, p.name]));
    const projectRows: UsageBreakdownRow[] = (byProject?.items ?? []).map((p) => ({
      id: p.ProjectID,
      name: projectLookup.get(p.ProjectID) ?? p.Name ?? p.ProjectID,
      href: `/app/observe/projects/${p.ProjectID}`,
      calls: p.Requests,
      tokens: p.TotalTokens,
      cost: microsToUsd(p.CostMicros),
    }));

    const teamLookup = new Map(teamList.items.map((t) => [t.id, t.name]));
    const teamRows: UsageBreakdownRow[] = (byTeam?.items ?? []).map((t) => ({
      id: t.TeamID,
      name: teamLookup.get(t.TeamID) ?? t.Name ?? t.TeamID,
      href: `/app/observe/teams/${t.TeamID}`,
      calls: t.Requests,
      tokens: t.TotalTokens,
      cost: microsToUsd(t.CostMicros),
    }));

    const userLookup = new Map(memberList.items.map((m) => [m.user.id, m.user.display_name || m.user.email]));
    const userRows: UsageBreakdownRow[] = (byUser?.items ?? []).map((u) => ({
      id: u.UserID,
      name: userLookup.get(u.UserID) ?? u.DisplayName ?? u.Email ?? u.UserID,
      href: `/app/observe/users/${u.UserID}`,
      calls: u.Requests,
      tokens: u.TotalTokens,
      cost: microsToUsd(u.CostMicros),
    }));

    return (
      <div className="p-4 sm:p-6">
        <PageShell
          title={displayName}
          description={`Model Telemetry & Attribution for ${modelSlug} (${catalogEntry?.provider_kind ?? "All Providers"})`}
        >
          <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
            <StatCard label="Total Requests" value={formatCompact(totalRequests)} />
            <StatCard label="Total Tokens" value={formatCompact(totalTokens)} />
            <StatCard label="Estimated Spend" value={formatUsd(totalCost)} />
            <StatCard label="P95 Latency" value={p95Latency ? formatMs(p95Latency) : "—"} />
            <StatCard
              label="Error Rate"
              value={formatPercent(errorRate, 2)}
              tone={errorRate >= 0.05 ? "danger" : errorRate >= 0.02 ? "warning" : "success"}
            />
          </section>

          <div className="grid gap-6 md:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle>Projects</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={breakdownColumns} rows={projectRows} rowKey="id" pageSize={5} emptyMessage="No project usage for this model." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Teams</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={breakdownColumns} rows={teamRows} rowKey="id" pageSize={5} emptyMessage="No team usage for this model." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Active Employees</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={breakdownColumns} rows={userRows} rowKey="id" pageSize={5} emptyMessage="No employee usage for this model." />
              </CardContent>
            </Card>
          </div>
        </PageShell>
      </div>
    );
  } catch (err) {
    notFound();
  }
}
