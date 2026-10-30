import { Suspense } from "react";
import Link from "next/link";
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  BarChart3,
  CheckCircle,
  Clock,
  Cpu,
  CreditCard,
  ExternalLink,
  Flame,
  Globe,
  HelpCircle,
  History,
  KeyRound,
  Layers,
  PieChart,
  Server,
  ShieldAlert,
  ShieldCheck,
  Sparkles,
  Users,
  XCircle,
  Zap,
} from "lucide-react";
import type { Alert, SecurityEvent } from "@ai-governance/api-client";

import {
  AreaChart,
  BarChart,
  DonutChart,
  LineChart,
  chartColor,
} from "@/components/charts";
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
  CategoryDatum,
  StatusTone,
  TrendDirection,
} from "@/types";
import { isDateRange, type DateRange } from "@/lib/date-range";
import { cn } from "@/lib/utils";
import { HeaderActions } from "./header-actions";
import { AnalyticsSuite } from "./analytics-suite";

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
      className={cn(
        "group flex flex-col sm:flex-row gap-4 sm:items-center justify-between p-4 rounded-xl border transition-all duration-200",
        isCompleted
          ? "bg-card/40 border-border/40 hover:border-border/70"
          : "bg-card/80 border-border hover:border-primary/50 hover:shadow-xs"
      )}
    >
      <div className="flex items-start gap-3.5 min-w-0 flex-1">
        <div
          className={cn(
            "mt-0.5 shrink-0 size-7 rounded-lg flex items-center justify-center text-xs font-semibold transition-colors",
            isCompleted
              ? "bg-emerald-500/15 text-emerald-400 border border-emerald-500/30"
              : "bg-muted text-foreground border border-border"
          )}
        >
          {isCompleted ? <CheckCircle className="size-4 text-emerald-400" /> : num}
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h4
              className={cn(
                "text-sm font-medium truncate",
                isCompleted ? "text-muted-foreground line-through decoration-muted-foreground/40" : "text-foreground"
              )}
            >
              {title}
            </h4>
            {isCompleted && (
              <span className="text-xs uppercase font-semibold tracking-wider px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Done
              </span>
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{desc}</p>
        </div>
      </div>
      <Link
        href={href}
        className={cn(
          "inline-flex items-center justify-center gap-1.5 shrink-0 text-xs font-medium px-3 py-1.5 rounded-lg transition-all",
          isCompleted
            ? "text-muted-foreground hover:text-foreground bg-secondary/40 hover:bg-secondary/70 border border-transparent"
            : "text-primary-foreground bg-primary hover:bg-primary/90 shadow-xs"
        )}
      >
        <span>{cta}</span>
        <ArrowRight className="size-3" />
      </Link>
    </div>
  );
}

interface SummaryTileProps {
  label: string;
  value: number;
  href: string;
  icon: React.ReactNode;
  accent: string;
}

function SummaryTile({ label, value, href, icon, accent }: SummaryTileProps) {
  return (
    <Link
      href={href}
      className="group flex items-center justify-between p-2.5 rounded-lg border border-border/40 bg-muted/20 hover:bg-muted/50 hover:border-border transition-all duration-150 outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      <div className="flex items-center gap-3 min-w-0">
        <span className={cn("flex size-7 items-center justify-center rounded-md text-xs", accent)}>
          {icon}
        </span>
        <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors truncate">
          {label}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold tabular-nums text-foreground">{value.toLocaleString("en-US")}</span>
        <ExternalLink className="size-3 text-muted-foreground/40 group-hover:text-muted-foreground transition-colors" />
      </div>
    </Link>
  );
}

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

  const providerCount = accounts?.items?.length ?? 0;
  const projectCount = projects?.items?.length ?? 0;
  const credentialCount = credentials?.items?.length ?? 0;
  const modelAccessCount = modelAccess?.items?.length ?? 0;

  const completedSteps = [hasProvider, hasProject, hasCredential, hasAccess, hasTraffic].filter(Boolean).length;
  const totalSetupSteps = 5;
  const isOperational = completedSteps === totalSetupSteps;

  if (isOperational) {
    return (
      <section aria-labelledby="operational-heading" className="p-4 sm:p-5 rounded-2xl border border-border/80 bg-linear-to-r from-card/90 via-card/60 to-card/40 backdrop-blur-xs shadow-xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="size-9 rounded-xl bg-emerald-500/15 text-emerald-400 border border-emerald-500/25 flex items-center justify-center shrink-0">
              <CheckCircle className="size-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 id="operational-heading" className="text-sm font-semibold text-foreground tracking-tight">
                  Workspace Operational
                </h2>
                <span className="text-xs font-semibold tracking-wider uppercase px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  Ready
                </span>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                All 5 milestone setup steps completed. Traffic routing, security, and governance policies are active.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 sm:gap-3 text-xs text-muted-foreground">
            <Link href="/app/platform/provider-accounts" className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/40 hover:bg-muted transition-colors">
              <Server className="size-3.5 text-blue-400" />
              <span>{providerCount} providers</span>
            </Link>
            <Link href="/app/organization/projects" className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/40 hover:bg-muted transition-colors">
              <Layers className="size-3.5 text-purple-400" />
              <span>{projectCount} projects</span>
            </Link>
            <Link href="/app/platform/gateway-credentials" className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/40 hover:bg-muted transition-colors">
              <KeyRound className="size-3.5 text-amber-400" />
              <span>{credentialCount} keys</span>
            </Link>
            <Link href="/app/govern/model-access" className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/40 hover:bg-muted transition-colors">
              <ShieldCheck className="size-3.5 text-emerald-400" />
              <span>{modelAccessCount} policies</span>
            </Link>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section
      aria-labelledby="onboarding-heading"
      className="p-5 sm:p-6 rounded-2xl border border-border/80 bg-linear-to-b from-card/90 via-card/60 to-card/40 backdrop-blur-xs shadow-xs"
    >
      <div className="space-y-6">
        <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-border/40">
          <div>
            <div className="flex items-center gap-2">
              <div className="size-2 rounded-full bg-blue-500 animate-pulse" />
              <h2 id="onboarding-heading" className="text-base font-semibold tracking-tight text-foreground">
                Get your AI gateway ready
              </h2>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
                {completedSteps} of {totalSetupSteps} completed
              </span>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Complete the essential setup steps to start routing AI requests, managing model access, and tracking costs.
            </p>
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <div className="h-2 w-36 sm:w-48 bg-muted/60 rounded-full overflow-hidden p-0.5 border border-border/40">
              <div
                className="h-full bg-linear-to-r from-blue-500 to-indigo-500 rounded-full transition-all duration-500"
                style={{ width: `${(completedSteps / totalSetupSteps) * 100}%` }}
                role="progressbar"
                aria-valuenow={completedSteps}
                aria-valuemin={0}
                aria-valuemax={totalSetupSteps}
                aria-label="Setup progress"
              />
            </div>
            <span className="text-xs font-semibold tabular-nums text-foreground">
              {Math.round((completedSteps / totalSetupSteps) * 100)}%
            </span>
          </div>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_300px] gap-6 items-start">
          <div className="space-y-2.5" role="list" aria-label="Setup steps">
            <SetupStep
              num={1}
              title="Connect an AI provider"
              desc="Connect upstream providers such as OpenAI, Anthropic, or Google Gemini."
              href="/app/platform/provider-accounts"
              cta="Connect provider"
              status={hasProvider ? "completed" : "not-started"}
            />
            <SetupStep
              num={2}
              title="Create a project"
              desc="Organize applications and attribute gateway usage to individual project workspaces."
              href="/app/organization/projects"
              cta="Create project"
              status={hasProject ? "completed" : "not-started"}
            />
            <SetupStep
              num={3}
              title="Create a gateway credential"
              desc="Generate secure API keys for authenticating downstream application requests."
              href="/app/platform/gateway-credentials"
              cta="Create credential"
              status={hasCredential ? "completed" : "not-started"}
            />
            <SetupStep
              num={4}
              title="Configure model access"
              desc="Define governance access policies restricting models, capabilities, and budgets."
              href="/app/govern/model-access"
              cta="Configure access"
              status={hasAccess ? "completed" : "not-started"}
            />
            <SetupStep
              num={5}
              title="Send your first request"
              desc="Route a test prompt through the gateway endpoint to verify telemetry flow."
              href="/app/developer/quick-start"
              cta="Open quick start"
              status={hasTraffic ? "completed" : "not-started"}
            />
          </div>

          <div className="rounded-xl border border-border/60 bg-card/60 p-4 lg:sticky lg:top-24 space-y-4">
            <div>
              <h3 className="font-semibold text-xs text-foreground uppercase tracking-wider">Workspace summary</h3>
              <p className="text-xs text-muted-foreground mt-0.5">Live resource inventory</p>
            </div>
            <div className="space-y-2">
              <SummaryTile
                label="Connected providers"
                value={providerCount}
                href="/app/platform/provider-accounts"
                icon={<Server className="size-3.5 text-blue-400" />}
                accent="bg-blue-500/10"
              />
              <SummaryTile
                label="Configured projects"
                value={projectCount}
                href="/app/organization/projects"
                icon={<Layers className="size-3.5 text-purple-400" />}
                accent="bg-purple-500/10"
              />
              <SummaryTile
                label="Active credentials"
                value={credentialCount}
                href="/app/platform/gateway-credentials"
                icon={<KeyRound className="size-3.5 text-amber-400" />}
                accent="bg-amber-500/10"
              />
              <SummaryTile
                label="Model access rules"
                value={modelAccessCount}
                href="/app/govern/model-access"
                icon={<ShieldCheck className="size-3.5 text-emerald-400" />}
                accent="bg-emerald-500/10"
              />
              <div className="flex items-center justify-between p-2.5 rounded-lg border border-border/40 bg-muted/20">
                <div className="flex items-center gap-3">
                  <span className="flex size-7 items-center justify-center rounded-md text-xs bg-sky-500/10">
                    <Activity className="size-3.5 text-sky-400" />
                  </span>
                  <span className="text-xs font-medium text-muted-foreground">Gateway requests</span>
                </div>
                <span className="text-sm font-semibold tabular-nums text-foreground">{requestCount.toLocaleString("en-US")}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
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
  accentClass?: string;
}

function ModernKpiCard({
  label,
  value,
  subtitle,
  icon: Icon,
  tone,
  delta,
  accentClass = "text-blue-400",
}: ModernKpiCardProps) {
  return (
    <div className="flex flex-col justify-between p-4 rounded-2xl border border-border/50 bg-card/80 hover:border-primary/40 hover:-translate-y-0.5 transition-all duration-200 shadow-xs group relative overflow-hidden">
      <div>
        <div className="flex items-center justify-between gap-2 mb-2">
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground group-hover:text-foreground transition-colors truncate">
            {label}
          </span>
          <div className={cn("size-7 rounded-lg bg-primary/10 flex items-center justify-center border border-primary/20", accentClass)}>
            <Icon className="size-3.5" />
          </div>
        </div>
        <p
          className={cn(
            "text-3xl font-bold tracking-tight tabular-nums",
            tone === "danger"
              ? "text-rose-500"
              : tone === "warning"
              ? "text-amber-500"
              : tone === "success"
              ? "text-emerald-500"
              : "text-foreground"
          )}
        >
          {value}
        </p>
      </div>

      <div className="flex items-center justify-between gap-2 mt-2 pt-2 border-t border-border/30 text-xs text-muted-foreground">
        <span>{subtitle || "Current window"}</span>
        {delta && (
          <span
            className={cn(
              "font-medium",
              delta.direction === "up" ? "text-emerald-400" : delta.direction === "down" ? "text-rose-400" : "text-muted-foreground"
            )}
          >
            {formatDelta(delta.fraction)}
          </span>
        )}
      </div>
    </div>
  );
}

async function KpiRowWidget({ timeParams }: { timeParams: TimeRangeQuery }) {
  const client = await getApiClient();
  const [summary, latency, series, activeUsersResp] = await Promise.all([
    safeFetch(() => client.usage.summary(timeParams), "summary"),
    safeFetch(() => client.usage.latency(timeParams), "latency"),
    safeFetch(() => client.usage.timeseries(timeParams), "timeseries"),
    safeFetch(() => client.activity.activeUsers(), "activeUsers"),
  ]);

  const totalRequests = summary?.Requests;
  const totalTokens = summary?.TotalTokens;
  const totalCostMicros = summary?.TotalCostMicros;
  const errors = summary?.Errors;

  const totalSpend = totalCostMicros != null ? microsToUsd(totalCostMicros) : null;
  const errorRate = totalRequests != null && totalRequests > 0 && errors != null
    ? errors / totalRequests
    : totalRequests === 0
    ? 0
    : null;

  const overall = latency?.items?.find((row) => !row.Provider && !row.Model);
  const p95 = overall?.P95TotalMs;
  const p95ttft = overall?.P95TTFTMs;
  const hasTTFT = p95ttft != null && p95ttft > 0;
  const hasP95 = p95 != null && p95 > 0;

  const latencyLabel = "P95 Latency";
  const latencyValue = latency === null
    ? "—"
    : hasP95
    ? formatMs(p95)
    : hasTTFT
    ? formatMs(p95ttft)
    : (totalRequests ?? 0) === 0
    ? "0 ms"
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
    <div className="grid gap-3.5 grid-cols-2 md:grid-cols-3 xl:grid-cols-6" role="region" aria-label="Key metrics">
      <ModernKpiCard
        label="Total requests"
        value={totalRequests != null ? formatCompact(totalRequests) : "—"}
        subtitle="Gateway calls"
        icon={Activity}
        delta={requestTrend}
        accentClass="text-sky-400 bg-sky-500/10"
      />
      <ModernKpiCard
        label="Total tokens"
        value={totalTokens != null ? formatCompact(totalTokens) : "—"}
        subtitle="In / Out / Cached"
        icon={Cpu}
        accentClass="text-purple-400 bg-purple-500/10"
      />
      <ModernKpiCard
        label="Estimated spend"
        value={totalSpend != null ? formatUsd(totalSpend) : "—"}
        subtitle="Model billing"
        icon={CreditCard}
        delta={spendTrend}
        accentClass="text-emerald-400 bg-emerald-500/10"
      />
      <ModernKpiCard
        label="Error rate"
        value={errorRate != null ? formatPercent(errorRate, 2) : "—"}
        subtitle="4xx & 5xx failures"
        icon={AlertTriangle}
        tone={errorRate != null ? errorTone(errorRate) : undefined}
        accentClass="text-amber-400 bg-amber-500/10"
      />
      <ModernKpiCard
        label={latencyLabel}
        value={latencyValue}
        subtitle={hasTTFT ? `P95 TTFT: ${formatMs(p95ttft)}` : "Roundtrip duration"}
        icon={Clock}
        accentClass="text-indigo-400 bg-indigo-500/10"
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
        subtitle="Unique consumers"
        icon={Users}
        accentClass="text-teal-400 bg-teal-500/10"
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Chart & Data Widgets
// ---------------------------------------------------------------------------

async function AnalyticsSuiteWidget({ timeParams }: { timeParams: TimeRangeQuery }) {
  const client = await getApiClient();
  const [series, summary, byProvider] = await Promise.all([
    safeFetch(() => client.usage.timeseries(timeParams), "timeseries"),
    safeFetch(() => client.usage.summary(timeParams), "summary"),
    safeFetch(() => client.usage.byProvider(timeParams), "byProvider"),
  ]);

  const requestSeries = (series?.items || []).map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: point.Requests,
  }));

  const spendSeries = (series?.items || []).map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: microsToUsd(point.CostMicros),
  }));

  const tokenData = [
    { label: "Input", value: summary?.InputTokens ?? 0 },
    { label: "Output", value: summary?.OutputTokens ?? 0 },
    { label: "Cached", value: summary?.CachedTokens ?? 0 },
  ].filter((d) => d.value > 0);

  const topProviders: CategoryDatum[] = [...(byProvider?.items || [])]
    .sort((a, b) => b.Requests - a.Requests)
    .slice(0, 5)
    .map((row) => ({ label: formatProviderName(row.Provider), value: row.Requests }));

  return (
    <AnalyticsSuite
      requestSeries={requestSeries}
      spendSeries={spendSeries}
      tokenData={tokenData}
      providerData={topProviders}
      totalRequests={summary?.Requests ?? 0}
      totalSpend={summary?.TotalCostMicros != null ? microsToUsd(summary.TotalCostMicros) : 0}
      totalTokens={summary?.TotalTokens ?? 0}
      cachedTokens={summary?.CachedTokens ?? 0}
    />
  );
}

async function ModelsWidget({ timeParams }: { timeParams: TimeRangeQuery }) {
  const client = await getApiClient();
  const byModel = await safeFetch(() => client.usage.byModel(timeParams), "byModel");

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

  const modelRows: ModelRow[] = (byModel?.items || [])
    .map((r) => ({
      id: r.Model,
      model: r.Model,
      requests: r.Requests,
      cost: microsToUsd(r.CostMicros),
      tokens: r.TotalTokens,
    }))
    .sort((a, b) => b.requests - a.requests)
    .slice(0, 5);

  if (modelRows.length === 0) {
    return (
      <EmptyVisualCard
        icon={Sparkles}
        title="No model usage recorded"
        description="Track latency, token consumption, and dollar volume broken down by individual model family."
        actionHref="/app/govern/model-access"
        actionLabel="View allowed model registry"
        ghostType="list"
      />
    );
  }

  return <DataTable columns={modelColumns} rows={modelRows} rowKey="id" emptyMessage="No model usage recorded yet." />;
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
      <EmptyVisualCard
        icon={ShieldCheck}
        title="All Systems Operational"
        description="Zero active incidents or security anomalies detected across the gateway perimeter."
        actionHref="/app/security/alerts"
        actionLabel="View alert policies"
        ghostType="shield"
        iconAccent="text-emerald-400 bg-emerald-500/10 border-emerald-500/25 shadow-xs"
      />
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
  const recentRequests = await safeFetch(() => client.usage.requests({ limit: 8 }), "recentRequests");

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
    { key: "user", header: "Consumer", primary: "user" },
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
    user: r.UserID || r.APICredentialID || "System",
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
  icon: Icon,
  title,
  subtitle,
  badge,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle?: string;
  badge?: string;
}) {
  return (
    <div className="flex items-center justify-between gap-3 pb-3 mb-4 border-b border-border/40">
      <div className="flex items-center gap-2.5">
        <div className="size-6 rounded-md bg-muted/40 border border-border/50 flex items-center justify-center text-muted-foreground">
          <Icon className="size-3.5" />
        </div>
        <div>
          <h2 className="text-base font-semibold tracking-tight text-foreground">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </div>
      {badge && (
        <span className="text-xs font-mono uppercase tracking-wider px-2.5 py-0.5 rounded-full bg-muted text-muted-foreground border border-border/60">
          {badge}
        </span>
      )}
    </div>
  );
}

function WidgetSkeleton() {
  return (
    <div className="animate-pulse bg-card/60 border border-border/40 rounded-xl h-55 w-full flex items-center justify-center">
      <div className="size-6 rounded-full border-2 border-border border-t-primary animate-spin" />
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
  const range: DateRange = isDateRange(raw) ? raw : "30d";

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
        description="Real-time observability, spending metrics, and governance enforcement across your AI workloads."
        actions={<HeaderActions lastUpdated={new Date()} />}
      >
        <div className="space-y-6 pb-12">
          {/* Onboarding / Operational Roadmap Hub */}
          <Suspense fallback={<WidgetSkeleton />}>
            <OnboardingWidget timeParams={timeParams} />
          </Suspense>

          {/* Primary Operational Pulse (Key Metrics) */}
          <section aria-labelledby="kpi-heading" className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 id="kpi-heading" className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                Operational Telemetry
              </h2>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium">
                <span className="size-2 rounded-full bg-emerald-500 animate-pulse" />
                <span>Telemetry active</span>
              </div>
            </div>
            <Suspense fallback={<WidgetSkeleton />}>
              <KpiRowWidget timeParams={timeParams} />
            </Suspense>
          </section>

          {/* Zone 2: Unified Interactive Analytics Suite */}
          <Suspense fallback={<WidgetSkeleton />}>
            <AnalyticsSuiteWidget timeParams={timeParams} />
          </Suspense>

          {/* Zone 3: Infrastructure Health & Model Allocation */}
          <div className="grid gap-5 lg:grid-cols-2">
            <div className="space-y-5 flex flex-col">
              <section className="bg-card/75 border border-border/70 rounded-2xl p-5 shadow-2xs flex-1 flex flex-col" aria-labelledby="health-heading">
                <SectionHeader icon={Server} title="Provider Health" subtitle="Upstream endpoint availability probes" />
                <div className="flex-1 flex flex-col justify-center min-h-0">
                  <Suspense fallback={<WidgetSkeleton />}>
                    <HealthWidget />
                  </Suspense>
                </div>
              </section>

              <section className="bg-card/75 border border-border/70 rounded-2xl p-5 shadow-2xs flex-1 flex flex-col" aria-labelledby="alerts-heading">
                <SectionHeader icon={ShieldAlert} title="Actionable Alerts" subtitle="Active anomalies and platform incidents" />
                <div className="flex-1 flex flex-col justify-center min-h-0">
                  <Suspense fallback={<WidgetSkeleton />}>
                    <AlertsWidget />
                  </Suspense>
                </div>
              </section>
            </div>

            <section className="bg-card/75 border border-border/70 rounded-2xl p-5 shadow-2xs flex flex-col" aria-labelledby="model-usage-heading">
              <SectionHeader icon={Zap} title="Model Family Allocation" subtitle="Most heavily utilized LLMs by tokens and cost" />
              <div className="flex-1 min-h-0">
                <Suspense fallback={<WidgetSkeleton />}>
                  <ModelsWidget timeParams={timeParams} />
                </Suspense>
              </div>
            </section>
          </div>

          {/* Live Stream: Recent Requests */}
          <section className="bg-card/75 border border-border/70 rounded-2xl p-5 shadow-2xs" aria-labelledby="recent-requests-heading">
            <SectionHeader icon={History} title="Recent Gateway Requests" subtitle="Live ledger of the latest proxied LLM transactions" badge="Streaming" />
            <Suspense fallback={<WidgetSkeleton />}>
              <RecentRequestsWidget />
            </Suspense>
          </section>
        </div>
      </PageShell>
    </div>
  );
}
