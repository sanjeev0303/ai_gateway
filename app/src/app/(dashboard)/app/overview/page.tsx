import { Suspense } from "react";
import { cookies } from "next/headers";
import Link from "next/link";
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  CheckCircle,
  Clock,
  Cpu,
  CreditCard,
  HelpCircle,
  History,
  Server,
  ShieldAlert,
  Users,
  XCircle,
} from "lucide-react";
import type { Alert } from "@ai-governance/api-client";

import {
  DataTable,
  PageShell,
  type Column,
} from "@/components/dashboard";
import {
  formatCompact,
  formatDateTime,
  formatDelta,
  formatMs,
  formatPercent,
  formatProviderName,
  formatUsd,
  microsToUsd,
} from "@/lib/format";

import { getApiClient } from "@/lib/session";
import type {
  TimeSeriesPoint,
  StatusTone,
  TrendDirection,
} from "@/types";
import { DEFAULT_RANGE, isDateRange, type DateRange } from "@/lib/date-range";
import { getDirectory, nameFor, nameMap } from "@/lib/directory";
import { cn } from "@/lib/utils";
import { GuillocheBand } from "@/components/brand/guilloche";
import { HeaderActions } from "./header-actions";
import { RankingCard, type RankRow } from "./ranking-card";
import { TrendChart, type TrendPoint } from "./trend-chart";
import { HERO_COOKIE, OverviewHero } from "./overview-hero";

type Trend = { direction: TrendDirection; fraction: number };

interface TimeRangeQuery {
  readonly from: Date;
  readonly to: Date;
}

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

async function safeFetch<T>(
  fn: () => Promise<T>,
  label: string
): Promise<T | null> {
  try {
    return await fn();
  } catch (error) {
    if (error instanceof Error && error.message === "NEXT_REDIRECT") {
      throw error;
    }
    console.error(`[OverviewPage] Failed to load ${label}:`, error);
    return null;
  }
}

// ---------------------------------------------------------------------------
// Design System & Bespoke Empty State Components
// ---------------------------------------------------------------------------

interface EmptyVisualCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  actionHref?: string;
  actionLabel?: string;
  ghostType?: "chart" | "donut" | "heatmap" | "bars" | "list" | "shield";
  iconAccent?: string;
}

function EmptyVisualCard({
  icon: Icon,
  title,
  description,
  actionHref,
  actionLabel,
  ghostType = "chart",
  iconAccent,
}: EmptyVisualCardProps) {
  return (
    <div className="relative flex flex-col items-center justify-center p-8 sm:p-12 text-center rounded-xl bg-muted/15 border border-dashed border-border/60 overflow-hidden min-h-55">
      {/* Ghost Visual Background */}
      <div className="absolute inset-0 pointer-events-none opacity-20 flex items-center justify-center overflow-hidden">
        {ghostType === "chart" && (
          <svg className="w-full h-28 stroke-border text-border" fill="none" viewBox="0 0 400 100" preserveAspectRatio="none">
            <line x1="0" y1="20" x2="400" y2="20" stroke="currentColor" strokeDasharray="4 4" strokeWidth="1" />
            <line x1="0" y1="50" x2="400" y2="50" stroke="currentColor" strokeDasharray="4 4" strokeWidth="1" />
            <line x1="0" y1="80" x2="400" y2="80" stroke="currentColor" strokeDasharray="4 4" strokeWidth="1" />
            <path d="M0,80 Q100,75 200,60 T400,30" stroke="currentColor" strokeWidth="1.5" />
          </svg>
        )}
        {ghostType === "donut" && (
          <div className="size-32 rounded-full border-4 border-dashed border-border" />
        )}
        {ghostType === "bars" && (
          <div className="flex items-end gap-3 h-20 w-3/4 justify-around">
            <div className="w-6 h-8 bg-border/40 rounded-t" />
            <div className="w-6 h-14 bg-border/40 rounded-t" />
            <div className="w-6 h-10 bg-border/40 rounded-t" />
            <div className="w-6 h-16 bg-border/40 rounded-t" />
          </div>
        )}
        {ghostType === "heatmap" && (
          <div className="grid grid-cols-12 gap-1 w-3/4">
            {Array.from({ length: 36 }).map((_, i) => (
              <div key={i} className="h-2.5 rounded-xs bg-border/30" />
            ))}
          </div>
        )}
        {ghostType === "list" && (
          <div className="w-3/4 space-y-2">
            <div className="h-3 bg-border/30 rounded w-full" />
            <div className="h-3 bg-border/30 rounded w-5/6" />
            <div className="h-3 bg-border/30 rounded w-2/3" />
          </div>
        )}
        {ghostType === "shield" && (
          <div className="size-28 rounded-full border border-dashed border-emerald-500/25 flex items-center justify-center">
            <div className="size-16 rounded-full border border-dashed border-emerald-500/35" />
          </div>
        )}
      </div>

      {/* Foreground Content */}
      <div className="relative z-10 flex flex-col items-center max-w-sm">
        <div className={cn("size-10 rounded-xl bg-card border border-border/80 shadow-xs flex items-center justify-center text-muted-foreground mb-3", iconAccent)}>
          <Icon className="size-5" />
        </div>
        <h3 className="text-sm font-medium text-foreground tracking-tight">{title}</h3>
        <p className="text-xs text-muted-foreground mt-1 leading-relaxed text-pretty">{description}</p>
        {actionHref && actionLabel && (
          <Link
            href={actionHref}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-primary hover:text-primary/80 mt-4 transition-colors group"
          >
            {actionLabel}
            <ArrowRight className="size-3 transition-transform group-hover:translate-x-0.5" />
          </Link>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Onboarding Step & Workspace Hub
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Widgets
// ---------------------------------------------------------------------------

async function OnboardingWidget({ timeParams }: { timeParams: TimeRangeQuery }) {
  const client = await getApiClient();
  const [accounts, projects, credentials, modelAccess, summary] = await Promise.all([
    safeFetch(() => client.providers.listAccounts(), "accounts"),
    safeFetch(() => client.projects.list(), "projects"),
    safeFetch(() => client.credentials.listGateway(), "credentials"),
    safeFetch(() => client.modelAccess.list(), "modelAccess"),
    safeFetch(() => client.usage.summary(timeParams), "summary"),
  ]);

  const hasProvider = (accounts?.items?.length ?? 0) > 0;
  const hasProject = (projects?.items?.length ?? 0) > 0;
  const hasCredential = (credentials?.items?.length ?? 0) > 0;
  const hasAccess = (modelAccess?.items?.length ?? 0) > 0;
  const requestCount = summary?.Requests ?? 0;
  const hasTraffic = requestCount > 0;
  const heroCookie = (await cookies()).get(HERO_COOKIE)?.value;

  const providerCount = accounts?.items?.length ?? 0;
  const credentialCount = credentials?.items?.length ?? 0;
  const modelAccessCount = modelAccess?.items?.length ?? 0;

  return (
    <OverviewHero
      stored={heroCookie}
      providerCount={providerCount}
      accessRuleCount={modelAccessCount}
      keyCount={credentialCount}
      requestCount={requestCount}
      setup={[
        { title: "Provider", cta: "Connect a provider", href: "/app/platform/provider-accounts", done: hasProvider },
        { title: "Project", cta: "Create a project", href: "/app/organization/projects", done: hasProject },
        { title: "Gateway key", cta: "Issue a gateway key", href: "/app/platform/gateway-credentials", done: hasCredential },
        { title: "Model access", cta: "Set model access", href: "/app/govern/model-access", done: hasAccess },
        { title: "First request", cta: "Send a request", href: "/app/developer/quick-start", done: hasTraffic },
      ]}
    />
  );
}

// ---------------------------------------------------------------------------
// KPI Deck
// ---------------------------------------------------------------------------

interface ModernKpiCardProps {
  label: string;
  value: string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
  tone?: StatusTone;
  delta?: Trend;
  /** Spend, latency and errors are better when they fall. */
  higherIsWorse?: boolean;
  /** Where to go to see what is behind the number. */
  href: string;
}

/** A figure with nothing behind it yet: no traffic in the range, or no data. */
function isEmptyFigure(value: string): boolean {
  return value === "—" || /^\$?0(\.0+)?(%| ms)?$/.test(value);
}

function ModernKpiCard({
  label,
  value,
  subtitle,
  icon: Icon,
  tone,
  delta,
  higherIsWorse = false,
  href,
}: ModernKpiCardProps) {
  let deltaTone = "text-muted-foreground";
  if (delta && delta.direction !== "flat") {
    const worse = higherIsWorse ? delta.direction === "up" : delta.direction === "down";
    deltaTone = worse ? "text-rose-700 dark:text-rose-400" : "text-emerald-700 dark:text-emerald-400";
  }
  // An empty figure is drawn as a ghost numeral: present, unlit, and quiet.
  const empty = isEmptyFigure(value);
  return (
    <Link
      href={href}
      aria-label={`${label}: ${value}. ${subtitle ?? ""} View details.`}
      className="group flex min-w-0 flex-col justify-between gap-3 bg-card p-4 transition-[background-color] duration-150 ease-(--ease-out-strong) hover:bg-accent/60 focus-visible:z-10 focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none"
    >
      <span className="flex items-center gap-2 text-[0.8125rem] font-medium text-muted-foreground">
        <Icon className="size-3.5 shrink-0 text-copper" />
        <span className="truncate">{label}</span>
      </span>
      <p
        data-slot="stat-value"
        className={cn(
          "font-display text-[1.75rem] leading-none tabular-nums",
          empty
            ? "text-foreground/25"
            : tone === "danger"
            ? "text-rose-700 dark:text-rose-400"
            : tone === "warning"
            ? "text-amber-700 dark:text-amber-400"
            : tone === "success"
            ? "text-emerald-700 dark:text-emerald-400"
            : "text-foreground"
        )}
      >
        {value}
      </p>
      {/* The ghost numeral already says "nothing yet"; a caption repeated in every cell is noise */}
      {!empty && (
        <span className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span className="truncate">{subtitle || "Current window"}</span>
          {delta && <span className={cn("font-medium tabular-nums", deltaTone)}>{formatDelta(delta.fraction)}</span>}
        </span>
      )}
    </Link>
  );
}

/** Plain words for the status codes a failed request records. */
function failureReason(code?: string | null): string {
  if (code === "429") return "rate limits";
  if (code === "403") return "policy or guardrail blocks";
  if (code === "402") return "budgets or provider credit";
  if (code === "401") return "invalid keys";
  if (code === "404") return "unavailable models";
  if (code === "400") return "rejected requests";
  if (code && code.startsWith("5")) return "provider errors";
  return "unclassified failures";
}

function topFailureReason(codes: readonly (string | null | undefined)[]): string | null {
  if (codes.length === 0) return null;
  const counts = new Map<string, number>();
  for (const code of codes) {
    const reason = failureReason(code);
    counts.set(reason, (counts.get(reason) ?? 0) + 1);
  }
  const [reason, count] = [...counts.entries()].sort((a, b) => b[1] - a[1])[0]!;
  return `${Math.round((count / codes.length) * 100)}% ${reason}`;
}

async function KpiRowWidget({ timeParams, rangeQuery }: { timeParams: TimeRangeQuery; rangeQuery: string }) {
  const client = await getApiClient();
  const [summary, latency, series, activeUsersResp, failed] = await Promise.all([
    safeFetch(() => client.usage.summary(timeParams), "summary"),
    safeFetch(() => client.usage.latency(timeParams), "latency"),
    safeFetch(() => client.usage.timeseries(timeParams), "timeseries"),
    safeFetch(() => client.activity.activeUsers(), "activeUsers"),
    safeFetch(() => client.usage.requests({ ...timeParams, status: "failed", limit: 200 }), "failedRequests"),
  ]);

  const failureSummary = topFailureReason((failed?.items ?? []).map((r) => r.ErrorCode));
  const withRange = (path: string, extra = "") => {
    const query = [rangeQuery, extra].filter(Boolean).join("&");
    return query ? `${path}?${query}` : path;
  };

  const totalRequests = summary?.Requests;
  const totalTokens = summary?.TotalTokens;
  const totalCostMicros = summary?.TotalCostMicros;
  const errors = summary?.Errors;

  const totalSpend = totalCostMicros != null ? microsToUsd(totalCostMicros) : null;
  const errorRate = totalRequests != null && totalRequests > 0 && errors != null
    ? errors / totalRequests
    : null;

  const overall = latency?.items?.find((row) => !row.Provider && !row.Model);
  const p95 = overall?.P95TotalMs;
  const p95ttft = overall?.P95TTFTMs;
  const hasTTFT = p95ttft != null && p95ttft > 0;
  const hasP95 = p95 != null && p95 > 0;

  const latencyLabel = "P95 latency";
  const latencyValue = latency === null
    ? "—"
    : hasP95
    ? formatMs(p95)
    : hasTTFT
    ? formatMs(p95ttft)
    : "—";


  const spendSeries = (series?.items || []).map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: microsToUsd(point.CostMicros),
  }));
  const requestSeries = (series?.items || []).map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: point.Requests,
  }));

  const spendTrend = series ? periodChange(spendSeries) : undefined;
  const requestTrend = series ? periodChange(requestSeries) : undefined;

  return (
    <div className="grid grid-cols-2 gap-px bg-border md:grid-cols-3 xl:grid-cols-6" role="region" aria-label="Key metrics">
      <ModernKpiCard
        label="Total requests"
        value={totalRequests != null ? formatCompact(totalRequests) : "—"}
        subtitle="Gateway calls"
        href={withRange("/app/observe/usage")}
        icon={Activity}
        delta={requestTrend}
      />
      <ModernKpiCard
        label="Total tokens"
        value={totalTokens != null ? formatCompact(totalTokens) : "—"}
        subtitle="In, out and cached"
        href={withRange("/app/observe/usage")}
        icon={Cpu}
      />
      <ModernKpiCard
        label="Spend"
        value={totalSpend != null ? formatUsd(totalSpend) : "—"}
        subtitle="Model billing"
        href={withRange("/app/observe/costs")}
        icon={CreditCard}
        delta={spendTrend}
        higherIsWorse
      />
      <ModernKpiCard
        label="Error rate"
        value={errorRate != null ? formatPercent(errorRate, 2) : "—"}
        subtitle={failureSummary ?? "No failures"}
        href={withRange("/app/observe/usage", "status=failed")}
        icon={AlertTriangle}
        tone={errorRate != null ? errorTone(errorRate) : undefined}
      />
      <ModernKpiCard
        label={latencyLabel}
        value={latencyValue}
        subtitle={hasTTFT ? `First token ${formatMs(p95ttft)}` : "Roundtrip duration"}
        href={withRange("/app/observe/models")}
        icon={Clock}
      />
      <ModernKpiCard
        label="Active users"
        value={
          summary?.ActiveUsers != null && summary.ActiveUsers > 0
            ? formatCompact(summary.ActiveUsers)
            : activeUsersResp === null
            ? "—"
            : activeUsersResp?.count != null
            ? formatCompact(activeUsersResp.count)
            : "0"
        }
        subtitle="Unique people"
        href={withRange("/app/observe/users")}
        icon={Users}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Chart & Data Widgets
// ---------------------------------------------------------------------------

async function UsageWidget({ timeParams }: { timeParams: TimeRangeQuery }) {
  const client = await getApiClient();
  const [series, byProvider, byModel] = await Promise.all([
    safeFetch(() => client.usage.timeseries(timeParams), "timeseries"),
    safeFetch(() => client.usage.byProvider(timeParams), "byProvider"),
    safeFetch(() => client.usage.byModel(timeParams), "byModel"),
  ]);

  const points: TrendPoint[] = (series?.items || []).map((point) => ({
    at: point.BucketStart,
    requests: point.Requests,
    spend: microsToUsd(point.CostMicros),
  }));
  const models: RankRow[] = [...(byModel?.items || [])]
    .sort((a, b) => b.Requests - a.Requests)
    .slice(0, 5)
    .map((r) => ({ name: r.Model, requests: r.Requests, spend: microsToUsd(r.CostMicros) }));
  const providers: RankRow[] = [...(byProvider?.items || [])]
    .sort((a, b) => b.Requests - a.Requests)
    .slice(0, 5)
    .map((r) => ({ name: formatProviderName(r.Provider), requests: r.Requests }));

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1.65fr)_minmax(0,1fr)]">
      <section aria-labelledby="traffic-heading" className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
        <div>
          <h2 id="traffic-heading" className="text-[0.9375rem] font-semibold tracking-[-0.01em]">
            Traffic
          </h2>
          <p className="text-xs text-muted-foreground">Requests and spend over the selected range</p>
        </div>
        <TrendChart points={points} />
      </section>
      <RankingCard models={models} providers={providers} />
    </div>
  );
}

async function HealthWidget() {
  const client = await getApiClient();
  const health = await safeFetch(() => client.providers.health(), "health");

  if (!health || !health.items || health.items.length === 0) {
    return (
      <EmptyVisualCard
        icon={Server}
        title="No providers connected"
        description="Active health probes inspect upstream latency and status every 10 seconds."
        actionHref="/app/platform/provider-accounts"
        actionLabel="Connect an upstream provider"
        ghostType="list"
      />
    );
  }

  return (
    <div className="flex flex-col gap-2.5">
      {health.items.map((h, i) => {
        let Icon = HelpCircle;
        let color = "text-muted-foreground";
        let statusBadge = "bg-muted text-muted-foreground border-border";
        if (h.status === "healthy" || h.status === "ok") {
          Icon = CheckCircle;
          color = "text-emerald-400";
          statusBadge = "bg-emerald-500/10 text-emerald-400 border-emerald-500/20";
        } else if (h.status === "degraded") {
          Icon = AlertTriangle;
          color = "text-amber-400";
          statusBadge = "bg-amber-500/10 text-amber-400 border-amber-500/20";
        } else if (h.status === "down" || h.status === "error") {
          Icon = XCircle;
          color = "text-rose-400";
          statusBadge = "bg-rose-500/10 text-rose-400 border-rose-500/20";
        }
        return (
          <div
            key={i}
            className="flex items-center justify-between p-3.5 border border-border/60 rounded-xl bg-card/60 text-sm transition-all hover:bg-muted/40 hover:border-border"
          >
            <div className="flex items-center gap-3">
              <span className="flex size-7 items-center justify-center rounded-lg bg-muted/40">
                <Icon className={`size-4 ${color}`} aria-hidden />
              </span>
              <div>
                <span className="font-semibold text-foreground text-sm">{formatProviderName(h.provider_kind)}</span>
                <p className="text-sm text-muted-foreground mt-0.5">Active health probe</p>
              </div>
            </div>
            <span className={cn("capitalize text-xs font-semibold px-2.5 py-0.5 rounded-full border", statusBadge)}>
              {h.status}
            </span>
          </div>
        );
      })}
    </div>
  );
}

async function AlertsWidget() {
  const client = await getApiClient();
  const alertsResp = await safeFetch(() => client.alerts.list(), "alerts");
  const alerts = alertsResp?.items || [];
  const triggeredAlerts = alerts.filter((a: Alert) => a.status === "active" && a.last_triggered_at != null);
  const activeAlerts = triggeredAlerts.length > 0 ? triggeredAlerts : alerts.filter((a: Alert) => a.status === "active").slice(0, 5);

  if (activeAlerts.length === 0) {
    return (
      // Say what the data shows, no more: nothing open, and where the rules live.
      <div className="flex flex-col items-center gap-1.5 py-6 text-center">
        <p className="text-sm font-semibold text-foreground">No open alerts</p>
        <p className="text-sm text-muted-foreground">Nothing has triggered an alert rule.</p>
        <Link
          href="/app/security/alerts"
          className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none"
        >
          View alert rules
          <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2.5">
      {activeAlerts.map((a: Alert, i: number) => {
        const isCritical = a.severity === "critical";
        const isWarning = a.severity === "warning";
        return (
          <Link
            key={i}
            href="/app/security/alerts"
            className="flex items-start gap-3 p-3.5 border border-border/60 rounded-xl bg-card/60 hover:bg-muted/40 hover:border-border transition-all"
          >
            <div
              className={cn(
                "size-8 rounded-lg flex items-center justify-center shrink-0 mt-0.5",
                isCritical
                  ? "bg-rose-500/10 text-rose-400"
                  : isWarning
                  ? "bg-amber-500/10 text-amber-400"
                  : "bg-blue-500/10 text-blue-400"
              )}
            >
              <ShieldAlert className="size-4.5" aria-hidden />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-semibold text-foreground truncate">{a.name}</p>
                <span className="text-sm text-muted-foreground tabular-nums shrink-0">
                  {a.last_triggered_at ? formatDateTime(a.last_triggered_at) : "Active"}
                </span>
              </div>
              <p className="text-sm text-muted-foreground mt-0.5 capitalize">{a.alert_type.replace("_", " ")} • {a.severity}</p>
            </div>
          </Link>
        );
      })}
    </div>
  );
}

async function RecentRequestsWidget() {
  const client = await getApiClient();
  const [recentRequests, directory] = await Promise.all([
    safeFetch(() => client.usage.requests({ limit: 8 }), "recentRequests"),
    getDirectory(),
  ]);
  const people = { ...nameMap(directory.users), ...nameMap(directory.serviceAccounts) };

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
    { key: "user", header: "Sent by", primary: "user" },
    { key: "provider", header: "Provider", primary: "provider" },
    { key: "model", header: "Model", primary: "model" },
    { key: "status", header: "Status", primary: "status" },
    { key: "tokens", header: "Tokens", primary: "tokens", align: "right", format: "compact" },
    { key: "cost", header: "Cost", primary: "cost", align: "right", format: "usd" },
    { key: "latency", header: "Latency", primary: "latency", align: "right" },
  ];

  const recentRows: RecentRow[] = (recentRequests?.items || []).map((r) => ({
    id: r.ID,
    timestamp: formatDateTime(r.CreatedAt),
    user: r.UserID ? nameFor(people, r.UserID) : r.APICredentialID ? "Service key" : "System",
    provider: formatProviderName(r.ProviderKind, r.Model),
    model: r.Model,
    status: r.Status,
    tokens: r.TotalTokens,
    cost: microsToUsd(r.CostMicros),
    latency: r.LatencyMs ? `${formatMs(r.LatencyMs)}` : "-",
  }));

  if (recentRows.length === 0) {
    return (
      <EmptyVisualCard
        icon={History}
        title="No recent gateway requests"
        description="Every request flowing through the AI gateway is captured in real-time with latency, tokens, cost, and policy evaluation."
        actionHref="/app/developer/quick-start"
        actionLabel="Explore Quick Start Guide"
        ghostType="list"
      />
    );
  }

  return (
    <DataTable
      columns={recentColumns}
      rows={recentRows}
      rowKey="id"
      emptyMessage="No recent requests found."
    />
  );
}

// ---------------------------------------------------------------------------
// Shell Wrapper & Section Header Helper
// ---------------------------------------------------------------------------

function SectionHeader({
  id,
  icon: Icon,
  title,
  subtitle,
  badge,
}: {
  id: string;
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle?: string;
  badge?: string;
}) {
  return (
    <div className="mb-4 flex items-start justify-between gap-3">
      <div className="flex items-start gap-2.5">
        <Icon className="mt-1 size-4 shrink-0 text-copper" />
        <div>
          <h2 id={id} className="text-[0.9375rem] font-semibold tracking-[-0.01em] text-foreground">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </div>
      {badge && (
        <span className="flex items-center gap-1.5 rounded-full bg-muted px-2.5 py-0.5 text-xs text-muted-foreground">
          <span className="size-1.5 rounded-full bg-emerald-500" aria-hidden />
          {badge}
        </span>
      )}
    </div>
  );
}

function WidgetSkeleton() {
  return <div className="h-55 w-full animate-pulse rounded-2xl border border-border bg-card/70" aria-hidden />;
}

/** The final layout with its values missing, so nothing jumps when data lands. */
function HeroSkeleton() {
  return <div className="h-[84px] w-full animate-pulse rounded-2xl border border-border bg-card/70 md:h-80 md:rounded-[28px]" aria-hidden />;
}

function KpiSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-px bg-border md:grid-cols-3 xl:grid-cols-6" aria-hidden>
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex h-[118px] flex-col justify-between bg-card p-4">
          <span className="h-3 w-20 animate-pulse rounded bg-muted" />
          <span className="h-6 w-16 animate-pulse rounded bg-muted" />
          <span className="h-3 w-24 animate-pulse rounded bg-muted" />
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page Route
// ---------------------------------------------------------------------------

export default async function OverviewPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : DEFAULT_RANGE;

  const RANGE_MS: Record<DateRange, number> = {
    "24h": 24 * 60 * 60 * 1000,
    "7d": 7 * 24 * 60 * 60 * 1000,
    "30d": 30 * 24 * 60 * 60 * 1000,
    "90d": 90 * 24 * 60 * 60 * 1000,
  };
  const to = new Date();
  const from = new Date(to.getTime() - RANGE_MS[range]);
  const timeParams: TimeRangeQuery = { from, to };

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Overview"
        description="Traffic, spend and enforcement across every model request in your workspace."
        actions={<HeaderActions lastUpdated={new Date()} />}
      >
        <div className="space-y-6 pb-12">
          {/* Health at a glance: one ledger of the figures that matter */}
          <section aria-labelledby="kpi-heading" className="overflow-hidden rounded-2xl border border-border-strong bg-card">
            <GuillocheBand className="opacity-70" />
            <div className="flex items-center justify-between gap-3 px-5 pt-1 pb-3">
              <h2 id="kpi-heading" className="text-[0.9375rem] font-semibold tracking-[-0.01em] text-foreground">
                Health
              </h2>
            </div>
            <div className="border-t border-border">
              <Suspense fallback={<KpiSkeleton />}>
                <KpiRowWidget timeParams={timeParams} rangeQuery={range === DEFAULT_RANGE ? "" : `range=${range}`} />
              </Suspense>
            </div>
          </section>

          {/* What needs attention */}
          <div className="grid gap-5 lg:grid-cols-2">
            <section className="flex flex-col rounded-2xl border border-border bg-card p-5" aria-labelledby="alerts-heading">
              <SectionHeader id="alerts-heading" icon={ShieldAlert} title="Alerts" subtitle="Active anomalies and platform incidents" />
              <div className="flex min-h-0 flex-1 flex-col justify-center">
                <Suspense fallback={<WidgetSkeleton />}>
                  <AlertsWidget />
                </Suspense>
              </div>
            </section>
            <section className="flex flex-col rounded-2xl border border-border bg-card p-5" aria-labelledby="health-heading">
              <SectionHeader id="health-heading" icon={Server} title="Provider health" subtitle="Latest availability probe per upstream" />
              <div className="flex min-h-0 flex-1 flex-col justify-center">
                <Suspense fallback={<WidgetSkeleton />}>
                  <HealthWidget />
                </Suspense>
              </div>
            </section>
          </div>

          {/* Setup until it is done, then the gateway's stages at a glance */}
          <Suspense fallback={<HeroSkeleton />}>
            <OnboardingWidget timeParams={timeParams} />
          </Suspense>

          <Suspense fallback={<WidgetSkeleton />}>
            <UsageWidget timeParams={timeParams} />
          </Suspense>

          <section className="rounded-2xl border border-border bg-card p-5" aria-labelledby="recent-requests-heading">
            <SectionHeader id="recent-requests-heading" icon={History} title="Recent requests" subtitle="The latest calls through the gateway" badge="Live" />
            <Suspense fallback={<WidgetSkeleton />}>
              <RecentRequestsWidget />
            </Suspense>
          </section>
        </div>
      </PageShell>
    </div>
  );
}
