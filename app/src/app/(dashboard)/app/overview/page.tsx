import { type ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  CheckCircle,
  HelpCircle,
  KeyRound,
  Layers,
  Server,
  ShieldAlert,
  ShieldCheck,
  Users,
  XCircle,
} from "lucide-react";
import {
  AreaChart,
  BarChart,
  DonutChart,
  LineChart,
  chartColor,
} from "@/components/charts";
import {
  DataTable,
  EmptyState,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  formatCompact,
  formatMs,
  formatPercent,
  formatUsd,
  microsToUsd,
} from "@/lib/format";

import { getApiClient } from "@/lib/session";
import type {
  TimeSeriesPoint,
  CategoryDatum,
  StatusTone,
  TrendDirection,
} from "@/types";
import { isDateRange, type DateRange } from "@/lib/date-range";
import { HeaderActions } from "./header-actions";

type Trend = { direction: TrendDirection; fraction: number };

function periodChange(series: readonly TimeSeriesPoint[]): Trend | undefined {
  if (series.length < 4) return undefined;
  const half = Math.floor(series.length / 2);
  let prior = 0;
  let current = 0;
  for (const point of series.slice(0, half)) prior += point.value;
  for (const point of series.slice(half)) current += point.value;
  if (prior === 0) return undefined;
  const fraction = (current - prior) / prior;
  return {
    direction: fraction > 0 ? "up" : fraction < 0 ? "down" : "flat",
    fraction,
  };
}

function errorTone(rate: number): StatusTone {
  if (rate >= 0.05) return "danger";
  if (rate >= 0.02) return "warning";
  return "success";
}

interface ActivityHeatmapProps {
  series: { items: { BucketStart: string; Requests: number }[] };
}
function ActivityHeatmap({ series }: ActivityHeatmapProps) {
  if (!series || !series.items || series.items.length === 0) {
    return (
      <EmptyState
        title="No activity data"
        description="Heatmap will appear when traffic is recorded."
      />
    );
  }

  const maxReq = Math.max(...series.items.map((s) => s.Requests));

  return (
    <div className="flex flex-wrap gap-1 mt-2" role="img" aria-label="Activity heatmap">
      {series.items.map((s, i) => {
        const intensity = maxReq > 0 ? s.Requests / maxReq : 0;
        let bg = "bg-muted";
        if (intensity > 0) bg = "bg-blue-500/20";
        if (intensity > 0.3) bg = "bg-blue-500/40";
        if (intensity > 0.6) bg = "bg-blue-500/70";
        if (intensity > 0.8) bg = "bg-blue-500";

        return (
          <div
            key={i}
            title={`${new Date(s.BucketStart).toLocaleString()}: ${s.Requests} requests`}
            className={`w-3.5 h-3.5 sm:w-4 sm:h-4 rounded-[2px] ${bg} hover:ring-1 hover:ring-foreground/50 transition-all`}
          />
        );
      })}
    </div>
  );
}

interface ProviderHealthProps {
  health: { items: { provider_kind: string; status: string }[] };
}
function ProviderHealth({ health }: ProviderHealthProps) {
  if (!health?.items?.length)
    return (
      <EmptyState
        title="No providers connected"
        description="Configure a provider to see its health status here."
      />
    );

  return (
    <div className="flex flex-col gap-2">
      {health.items.map((h, i) => {
        let Icon = HelpCircle;
        let color = "text-muted-foreground";
        if (h.status === "healthy" || h.status === "ok") {
          Icon = CheckCircle;
          color = "text-emerald-500";
        } else if (h.status === "degraded") {
          Icon = AlertTriangle;
          color = "text-amber-500";
        } else if (h.status === "down" || h.status === "error") {
          Icon = XCircle;
          color = "text-red-500";
        }
        return (
          <div key={i} className="flex items-center justify-between p-3 border rounded-lg bg-card text-sm transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Icon className={`size-4 ${color}`} aria-hidden />
              <span className="font-medium truncate">{h.provider_kind || "Unknown"}</span>
            </div>
            <span className="text-muted-foreground capitalize text-xs font-medium">{h.status}</span>
          </div>
        );
      })}
    </div>
  );
}

interface ActionableAlertsProps {
  events: { title: string; severity: string; created_at: string }[];
}
function ActionableAlerts({ events }: ActionableAlertsProps) {
  if (!events || events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-6 text-center border rounded-xl bg-card min-h-[160px] h-full">
        <ShieldCheck className="size-8 text-emerald-500 mb-3" aria-hidden />
        <h3 className="font-medium text-sm">All Systems Operational</h3>
        <p className="text-xs text-muted-foreground mt-1 max-w-[200px]">No actionable alerts or active incidents.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {events.map((ev, i) => (
        <a key={i} href="/app/observe/security" className="flex items-start gap-3 p-3 border rounded-lg bg-card hover:bg-muted/50 transition-colors">
          <ShieldAlert className={`size-4 mt-0.5 shrink-0 ${ev.severity === 'critical' ? 'text-red-500' : ev.severity === 'warning' ? 'text-amber-500' : 'text-blue-500'}`} aria-hidden />
          <div className="min-w-0">
            <p className="text-sm font-medium truncate">{ev.title}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{new Date(ev.created_at).toLocaleString()}</p>
          </div>
        </a>
      ))}
    </div>
  );
}

interface ZeroTrafficStateData {
  accounts: { items: unknown[] };
  projects: { items: unknown[] };
  credentials: { items: unknown[] };
  modelAccess: { items: unknown[] };
  summary: { Requests: number; TotalCostMicros: number };
}

function ZeroTrafficState({
  accounts,
  projects,
  credentials,
  modelAccess,
  summary,
}: ZeroTrafficStateData) {
  const hasProvider = accounts?.items?.length > 0;
  const hasProject = projects?.items?.length > 0;
  const hasCredential = credentials?.items?.length > 0;
  const hasAccess = modelAccess?.items?.length > 0;

  const providerCount = accounts?.items?.length ?? 0;
  const projectCount = projects?.items?.length ?? 0;
  const credentialCount = credentials?.items?.length ?? 0;
  const modelAccessCount = modelAccess?.items?.length ?? 0;
  const requestCount = summary?.Requests ?? 0;

  const completedSteps = [hasProvider, hasProject, hasCredential, hasAccess].filter(Boolean).length;
  const totalSetupSteps = 5;

  return (
    <PageShell
      title="Overview"
      description="Monitor AI traffic, spending, performance, and governance across your organization."
      eyebrow={<span className="text-sm font-medium text-muted-foreground">Workspace / Overview</span>}
    >
      <div className="space-y-8">
        <section aria-labelledby="onboarding-heading">
          <div className="space-y-5">
            <header className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
              <div>
                <h2 id="onboarding-heading" className="text-lg font-semibold tracking-tight">Get your AI gateway ready</h2>
                <p className="text-muted-foreground text-sm">
                  Complete the essential setup steps to start routing AI requests and tracking usage.
                </p>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <div className="hidden sm:flex items-center gap-2 text-sm font-medium text-muted-foreground">
                  <span>Setup progress</span>
                  <span className="text-foreground font-semibold">{completedSteps}</span>
                  <span className="text-muted-foreground">/</span>
                  <span className="text-foreground font-semibold">{totalSetupSteps}</span>
                </div>
                <div className="hidden sm:block h-2 w-40 bg-muted rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all duration-300"
                    style={{ width: `${(completedSteps / totalSetupSteps) * 100}%` }}
                    role="progressbar"
                    aria-valuenow={completedSteps}
                    aria-valuemin={0}
                    aria-valuemax={totalSetupSteps}
                    aria-label="Setup progress"
                  />
                </div>
              </div>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-6 items-start">
              <div className="space-y-2" role="list" aria-label="Setup steps">
                <SetupStep
                  num={1}
                  title="Connect an AI provider"
                  desc="Connect an upstream provider such as OpenAI, Anthropic, or Gemini."
                  href="/app/platform/provider-accounts"
                  cta="Connect provider"
                  status={hasProvider ? "completed" : "not-started"}
                />
                <SetupStep
                  num={2}
                  title="Create a project"
                  desc="Organize applications and attribute gateway usage to individual projects."
                  href="/app/organization/projects"
                  cta="Create project"
                  status={hasProject ? "completed" : "not-started"}
                />
                <SetupStep
                  num={3}
                  title="Create a gateway credential"
                  desc="Generate a gateway credential for authenticating application requests."
                  href="/app/platform/gateway-credentials"
                  cta="Create credential"
                  status={hasCredential ? "completed" : "not-started"}
                />
                <SetupStep
                  num={4}
                  title="Configure model access"
                  desc="Choose which models and capabilities your organization can use."
                  href="/app/govern/model-access"
                  cta="Configure access"
                  status={hasAccess ? "completed" : "not-started"}
                />
                <SetupStep
                  num={5}
                  title="Send your first request"
                  desc="Use the gateway endpoint and credential to send a test request."
                  href="/app/developer/quick-start"
                  cta="Open quick start"
                  status="not-started"
                />
              </div>

              <div className="rounded-xl border bg-card p-5 lg:sticky lg:top-24">
                <h3 className="font-medium text-sm mb-4">Workspace summary</h3>
                <div className="space-y-3">
                  <MetricRow label="Connected providers" value={providerCount} href="/app/platform/provider-accounts" icon={<Server className="size-4" />} />
                  <MetricRow label="Configured projects" value={projectCount} href="/app/organization/projects" icon={<Layers className="size-4" />} />
                  <MetricRow label="Active credentials" value={credentialCount} href="/app/platform/gateway-credentials" icon={<KeyRound className="size-4" />} />
                  <MetricRow label="Model access policies" value={modelAccessCount} href="/app/govern/model-access" icon={<ShieldCheck className="size-4" />} />
                  <MetricRow label="Gateway requests" value={requestCount} icon={<Activity className="size-4" />} />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section aria-labelledby="empty-state-heading">
          <div className="rounded-xl border bg-card/30 flex flex-col items-center justify-center text-center p-8 lg:p-12 min-h-[220px]">
            <div className="size-12 rounded-full bg-blue-500/10 text-blue-500 flex items-center justify-center mb-5 shrink-0">
              <Activity className="size-6" aria-hidden />
            </div>
            <h3 id="empty-state-heading" className="text-lg font-medium tracking-tight">Your AI activity will appear here</h3>
            <p className="mt-2 text-sm text-muted-foreground max-w-md">
              Once your first request passes through the gateway, this area will show request volume, token usage, spend, errors, and latency for the selected period.
            </p>
            <div className="mt-6">
              <a href="/app/developer/quick-start" className="inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium border bg-background hover:bg-muted h-9 px-4 transition-colors">
                View quick start
              </a>
            </div>
          </div>
        </section>
      </div>
    </PageShell>
  );
}

interface SetupStepProps {
  num: number;
  title: string;
  desc: string;
  href: string;
  cta: string;
  status: "completed" | "not-started";
}
function SetupStep({ num, title, desc, href, cta, status }: SetupStepProps) {
  const isCompleted = status === "completed";
  return (
    <div
      role="listitem"
      className={`group flex flex-col sm:flex-row gap-4 sm:items-center justify-between p-4 rounded-xl border transition-all duration-150 ${isCompleted ? "bg-card border-border/50" : "bg-card hover:border-border"}`}
    >
      <div className="flex items-start gap-4 min-w-0 flex-1">
        <div className={`mt-0.5 shrink-0 size-7 rounded-full flex items-center justify-center text-xs font-medium ${isCompleted ? "bg-emerald-500/10 text-emerald-500" : "bg-muted text-muted-foreground"}`}>
          {isCompleted ? <CheckCircle className="size-4" /> : num}
        </div>
        <div className="min-w-0">
          <h4 className="text-sm font-medium text-foreground truncate">{title}</h4>
          <p className="text-xs text-muted-foreground mt-1 leading-relaxed">{desc}</p>
        </div>
      </div>
      <a
        href={href}
        className={`inline-flex items-center gap-1.5 shrink-0 text-sm font-medium px-3 py-1.5 rounded-md transition-colors ${isCompleted ? "text-muted-foreground hover:text-foreground" : "text-blue-500 hover:text-blue-400 bg-blue-500/10 hover:bg-blue-500/20"}`}
      >
        {cta}
      </a>
    </div>
  );
}

interface MetricRowProps {
  label: string;
  value: number;
  href?: string;
  icon?: ReactNode;
}
function MetricRow({ label, value, href, icon }: MetricRowProps) {
  const content = (
    <div className="flex items-center gap-3">
      {icon && <span className="flex shrink-0 size-7 items-center justify-center rounded-md bg-muted text-muted-foreground">{icon}</span>}
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground truncate">{label}</p>
        <p className="text-sm font-medium tabular-nums text-foreground">{value.toLocaleString()}</p>
      </div>
    </div>
  );
  if (href) {
    return (
      <a href={href} className="group block p-2 -mx-2 rounded-lg hover:bg-muted/50 transition-all outline-none focus-visible:ring-2 focus-visible:ring-primary">
        {content}
      </a>
    );
  }
  return <div className="p-2 -mx-2">{content}</div>;
}

interface TrafficStateProps {
  summary: {
    Requests: number;
    TotalCostMicros: number;
    Errors: number;
    TotalTokens: number;
    InputTokens: number;
    OutputTokens: number;
    CachedTokens: number;
  };
  latency: { items: { Provider?: string; Model?: string; P95TotalMs?: number; P95TTFTMs?: number }[] };
  series: { items: { BucketStart: string; Requests: number; CostMicros: number }[] };
  byProject: { items: unknown[] };
  byModel: { items: { Model: string; Requests: number; CostMicros: number; TotalTokens: number }[] };
  byProvider: { items: { Provider: string; Requests: number }[] };
  byTeam: { items: unknown[] };
  accounts: { items: unknown[] };
  health: { items: { provider_kind: string; status: string }[] };
  securityEvents: { items: { title: string; severity: string; created_at: string }[] };
  recentRequests: {
    items: {
      ID: string;
      CreatedAt: string;
      UserID?: string;
      APICredentialID?: string;
      ProviderKind: string;
      Model: string;
      Status: string;
      TotalTokens: number;
      CostMicros: number;
      LatencyMs: number;
    }[];
  };
  activeUsers: number | null;
  actions: ReactNode;
}

function TrafficState({
  summary,
  latency,
  series,
  byModel,
  byProvider,
  health,
  securityEvents,
  recentRequests,
  activeUsers,
  actions,
}: TrafficStateProps) {
  const totalSpend = microsToUsd(summary.TotalCostMicros);
  const errorRate = summary.Requests > 0 ? summary.Errors / summary.Requests : 0;
  const overall = latency.items.find((row) => !row.Provider && !row.Model);
  const p95 = overall?.P95TotalMs;
  const p95ttft = overall?.P95TTFTMs;
  const hasLatencyData = p95 !== undefined;

  const spendSeries = series.items.map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: microsToUsd(point.CostMicros),
  }));
  const requestSeries = series.items.map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: point.Requests,
  }));

  const spendTrend = periodChange(spendSeries);
  const requestTrend = periodChange(requestSeries);

  const stats = [
    {
      label: "Total requests",
      value: formatCompact(summary.Requests),
      sparkline: requestSeries,
      delta: requestTrend,
      icon: Activity,
    },
    {
      label: "Total tokens",
      value: formatCompact(summary.TotalTokens),
      icon: Layers,
    },
    {
      label: "Estimated AI spend",
      value: formatUsd(totalSpend),
      sparkline: spendSeries,
      delta: spendTrend ? { ...spendTrend, higherIsWorse: true } : undefined,
      icon: Server,
    },
    {
      label: "Error rate",
      value: formatPercent(errorRate, 2),
      tone: errorTone(errorRate) as StatusTone,
      icon: AlertTriangle,
    },
    {
      label: p95ttft !== undefined ? "P95 TTFT" : "P95 Latency",
      value: hasLatencyData ? formatMs(p95 ?? p95ttft ?? 0) : "n/a",
      icon: Activity,
    },
    {
      label: "Active users",
      value: activeUsers !== null ? formatCompact(activeUsers) : "n/a",
      icon: Users,
    },
  ];

  const tokenData = [
    { label: "Input", value: summary.InputTokens },
    { label: "Output", value: summary.OutputTokens },
    { label: "Cached", value: summary.CachedTokens },
  ].filter((d) => d.value > 0);

  const topProviders: CategoryDatum[] = [...byProvider.items]
    .sort((a, b) => b.Requests - a.Requests)
    .slice(0, 5)
    .map((row) => ({ label: row.Provider, value: row.Requests }));

  interface ModelRow {
    id: string;
    model: string;
    requests: number;
    cost: number;
    tokens: number;
  }
  const modelColumns: Column<ModelRow>[] = [
    { key: "model", header: "Model", primary: "model" },
    { key: "requests", header: "Reqs", primary: "requests", align: "right", format: "compact" },
    { key: "cost", header: "Spend", primary: "cost", align: "right", format: "usd" },
    { key: "tokens", header: "Tokens", primary: "tokens", align: "right", format: "compact" },
  ];
  const modelRows: ModelRow[] = byModel.items
    .map((r) => ({
      id: r.Model,
      model: r.Model,
      requests: r.Requests,
      cost: microsToUsd(r.CostMicros),
      tokens: r.TotalTokens,
    }))
    .sort((a, b) => b.requests - a.requests)
    .slice(0, 5);

  interface RecentRow {
    id: string;
    timestamp: string;
    user: string;
    provider: string;
    model: string;
    status: string;
    tokens: number;
    cost: number;
    latency: string;
  }
  const recentColumns: Column<RecentRow>[] = [
    { key: "timestamp", header: "Time", primary: "timestamp" },
    { key: "user", header: "User", primary: "user" },
    { key: "provider", header: "Provider", primary: "provider" },
    { key: "model", header: "Model", primary: "model" },
    { key: "status", header: "Status", primary: "status" },
    { key: "tokens", header: "Tokens", primary: "tokens", align: "right", format: "compact" },
    { key: "cost", header: "Cost", primary: "cost", align: "right", format: "usd" },
    { key: "latency", header: "Latency", primary: "latency", align: "right" },
  ];
  const recentRows: RecentRow[] = recentRequests.items.map((r) => ({
    id: r.ID,
    timestamp: new Date(r.CreatedAt).toLocaleTimeString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }),
    user: r.UserID || r.APICredentialID || "System",
    provider: r.ProviderKind,
    model: r.Model,
    status: r.Status,
    tokens: r.TotalTokens,
    cost: microsToUsd(r.CostMicros),
    latency: r.LatencyMs ? `${formatMs(r.LatencyMs)}` : "-",
  }));

  return (
    <PageShell
      title="Overview"
      description="Monitor AI traffic, spending, performance, and governance across your organization."
      eyebrow={<span className="text-sm font-medium text-muted-foreground">Workspace / Overview</span>}
      actions={actions}
    >
      <div className="space-y-8 pb-8">
        {/* KPI row */}
        <div className="grid gap-4 grid-cols-2 md:grid-cols-3 xl:grid-cols-6" role="region" aria-label="Key metrics">
          {stats.map((stat) => (
            <StatCard key={stat.label} {...stat} />
          ))}
        </div>

        {/* Primary analytics */}
        <div className="grid gap-6 lg:grid-cols-2">
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="request-volume-heading">
            <h2 id="request-volume-heading" className="text-sm font-medium tracking-tight">Request volume</h2>
            <LineChart data={requestSeries} color={chartColor(0)} formatValue={formatCompact} />
          </section>
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="spend-heading">
            <h2 id="spend-heading" className="text-sm font-medium tracking-tight">Estimated AI spend</h2>
            <AreaChart series={[{ label: "Spend", color: chartColor(1), data: spendSeries }]} formatValue={formatUsd} />
            <p className="text-xs text-muted-foreground">Costs are estimated from model pricing at request time. Actual provider invoices may differ.</p>
          </section>
        </div>

        {/* Token analytics and heatmap */}
        <div className="grid gap-6 lg:grid-cols-2">
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="token-composition-heading">
            <h2 id="token-composition-heading" className="text-sm font-medium tracking-tight">Token composition</h2>
            {tokenData.length > 0 ? (
              <DonutChart data={tokenData} centerLabel="tokens" />
            ) : (
              <EmptyState title="No token data" />
            )}
          </section>
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="activity-heatmap-heading">
            <h2 id="activity-heatmap-heading" className="text-sm font-medium tracking-tight">Activity heatmap</h2>
            <ActivityHeatmap series={series} />
            <p className="text-xs text-muted-foreground">Each cell represents a time bucket. Darker cells indicate higher request volume.</p>
          </section>
        </div>

        {/* Provider and model distribution */}
        <div className="grid gap-6 lg:grid-cols-2">
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="provider-dist-heading">
            <h2 id="provider-dist-heading" className="text-sm font-medium tracking-tight">Provider distribution</h2>
            {topProviders.length > 0 ? (
              <BarChart data={topProviders} formatValue={formatCompact} />
            ) : (
              <EmptyState title="No provider usage" />
            )}
          </section>
          <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="model-usage-heading">
            <h2 id="model-usage-heading" className="text-sm font-medium tracking-tight">Top models usage</h2>
            <DataTable columns={modelColumns} rows={modelRows} rowKey="id" emptyMessage="No model usage recorded yet." />
          </section>
        </div>

        {/* Health and Alerts */}
        <div className="grid gap-6 lg:grid-cols-2 items-start">
          <section className="space-y-4 bg-card border rounded-xl p-5 h-full" aria-labelledby="health-heading">
            <h2 id="health-heading" className="text-sm font-medium tracking-tight">Provider health</h2>
            <ProviderHealth health={health} />
          </section>
          <section className="space-y-4 bg-card border rounded-xl p-5 h-full" aria-labelledby="alerts-heading">
            <h2 id="alerts-heading" className="text-sm font-medium tracking-tight">Actionable alerts</h2>
            <ActionableAlerts events={securityEvents.items || []} />
          </section>
        </div>

        {/* Recent gateway requests */}
        <section className="space-y-4 bg-card border rounded-xl p-5" aria-labelledby="recent-requests-heading">
          <h2 id="recent-requests-heading" className="text-sm font-medium tracking-tight">Recent gateway requests</h2>
          <DataTable columns={recentColumns} rows={recentRows} rowKey="id" emptyMessage="No recent requests found." />
        </section>
      </div>
    </PageShell>
  );
}

export default async function OverviewPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const client = await getApiClient();

  const RANGE_MS: Record<DateRange, number> = {
    "24h": 24 * 60 * 60 * 1000,
    "7d": 7 * 24 * 60 * 60 * 1000,
    "30d": 30 * 24 * 60 * 60 * 1000,
    "90d": 90 * 24 * 60 * 60 * 1000,
  };
  const to = new Date();
  const from = new Date(to.getTime() - RANGE_MS[range]);
  const timeParams = { from, to };

  let summary: Awaited<ReturnType<typeof client.usage.summary>>;
  let latency: Awaited<ReturnType<typeof client.usage.latency>>;
  let series: Awaited<ReturnType<typeof client.usage.timeseries>>;
  let byProject: Awaited<ReturnType<typeof client.usage.byProject>>;
  let byModel: Awaited<ReturnType<typeof client.usage.byModel>>;
  let byProvider: Awaited<ReturnType<typeof client.usage.byProvider>>;
  let byTeam: Awaited<ReturnType<typeof client.usage.byTeam>>;
  let accounts: Awaited<ReturnType<typeof client.providers.listAccounts>>;
  let health: Awaited<ReturnType<typeof client.providers.health>>;
  let projectsList: Awaited<ReturnType<typeof client.projects.list>>;
  let credentialsList: Awaited<ReturnType<typeof client.credentials.listGateway>>;
  let modelAccessList: Awaited<ReturnType<typeof client.modelAccess.list>>;
  let securityEventsList: Awaited<ReturnType<typeof client.securityEvents.list>>;
  let recentRequests: Awaited<ReturnType<typeof client.usage.requests>>;
  let activeUsersResp: Awaited<ReturnType<typeof client.activity.activeUsers>> | null = null;

  try {
    [
      summary,
      latency,
      series,
      byProject,
      byModel,
      byProvider,
      byTeam,
      accounts,
      health,
      projectsList,
      credentialsList,
      modelAccessList,
      securityEventsList,
      recentRequests,
    ] = await Promise.all([
      client.usage.summary(timeParams),
      client.usage.latency(timeParams),
      client.usage.timeseries(timeParams),
      client.usage.byProject(timeParams),
      client.usage.byModel(timeParams),
      client.usage.byProvider(timeParams),
      client.usage.byTeam(timeParams),
      client.providers.listAccounts(),
      client.providers.health(),
      client.projects.list(),
      client.credentials.listGateway(),
      client.modelAccess.list(),
      client.securityEvents.list({ limit: 5 }),
      client.usage.requests({ limit: 8 }),
    ]);

    try {
      activeUsersResp = await client.activity.activeUsers();
    } catch {
      // ignore
    }
  } catch (error) {
    if (error instanceof Error && error.message === "NEXT_REDIRECT") {
      throw error;
    }
    console.error("[OverviewPage] Failed to load dashboard data:", error);
    throw error;
  }

  if (summary.Requests === 0 && summary.TotalCostMicros === 0) {
    return (
      <ZeroTrafficState
        accounts={accounts}
        projects={projectsList}
        credentials={credentialsList}
        modelAccess={modelAccessList}
        summary={summary}
      />
    );
  }

  return (
    <TrafficState
      summary={summary}
      latency={latency}
      series={series}
      byProject={byProject}
      byModel={byModel}
      byProvider={byProvider}
      byTeam={byTeam}
      accounts={accounts}
      health={health}
      securityEvents={securityEventsList}
      recentRequests={recentRequests}
      activeUsers={activeUsersResp?.count ?? null}
      actions={
        <HeaderActions lastUpdated={new Date()} />
      }
    />
  );
}