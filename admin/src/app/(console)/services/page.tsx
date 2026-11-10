import type { Metadata } from "next";
import { AlertTriangle, Clock, Inbox, RefreshCw } from "lucide-react";

import type { AdminServiceSample, AdminServices, AdminSystem } from "@ai-governance/api-client";
import { AreaChart } from "@ai-governance/console-ui/charts/area-chart";
import { chartColor } from "@ai-governance/console-ui/charts/utils";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { StatusBadge } from "@ai-governance/console-ui/dashboard/status-badge";
import { formatDateTime, formatNumber, formatProviderName, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { Ledger, LedgerCell, Panel, Register, RegisterEmpty, RegisterRow } from "../_components/ledger";
import { formatBytes, formatUptime, healthTone, serviceLabel } from "../_lib";

export const metadata: Metadata = { title: "Services · Platform console" };

// The page re-reads on every visit; the worker samples every 30 seconds.
export const dynamic = "force-dynamic";

const STALE_SECONDS = 5 * 60;

interface ServiceRow {
  id: string;
  name: string;
  error: string;
  status: string;
  statusTone: StatusTone;
  cpu: string;
  cpuSort: number;
  memory: string;
  memorySort: number;
  heap: string;
  goroutines: string;
  dbPool: string;
  uptime: string;
  checked: string;
}

const COLUMNS: Column<ServiceRow>[] = [
  { key: "name", header: "Service", primary: "name", secondary: "error", grow: true },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "cpu", header: "CPU", primary: "cpu", sort: "cpuSort", align: "right" },
  { key: "memory", header: "Memory", primary: "memory", sort: "memorySort", align: "right" },
  { key: "heap", header: "Go heap", primary: "heap", align: "right" },
  { key: "goroutines", header: "Goroutines", primary: "goroutines", align: "right" },
  { key: "pool", header: "DB connections", primary: "dbPool", align: "right" },
  { key: "uptime", header: "Uptime", primary: "uptime", align: "right" },
  { key: "checked", header: "Checked", primary: "checked" },
];

function num(value: unknown): number | undefined {
  return typeof value === "number" ? value : undefined;
}

function toRow(s: AdminServiceSample): ServiceRow {
  const pool = s.detail.db_pool as { acquired?: number; max?: number } | null | undefined;
  // Postgres and Redis report their own figures in detail rather than as a process.
  let memory = formatBytes(s.rss_bytes);
  if (s.service === "postgres") memory = `${formatBytes(num(s.detail.size_bytes))} on disk`;
  let dbPool = pool && pool.max ? `${pool.acquired ?? 0} of ${pool.max}` : "—";
  if (s.service === "postgres") dbPool = `${num(s.detail.connections) ?? "—"} open`;
  if (s.service === "redis") {
    const clients = num(s.detail.connected_clients);
    dbPool = clients !== undefined ? `${clients} clients` : "—";
  }

  return {
    id: s.service,
    name: serviceLabel(s.service),
    error: s.error ?? "",
    status: s.up ? "Up" : "Down",
    statusTone: s.up ? "success" : "danger",
    cpu: s.cpu_percent !== undefined ? `${s.cpu_percent.toFixed(1)}%` : "—",
    cpuSort: s.cpu_percent ?? -1,
    memory,
    memorySort: s.rss_bytes ?? num(s.detail.size_bytes) ?? -1,
    heap: formatBytes(s.heap_bytes),
    goroutines: s.goroutines !== undefined ? formatNumber(s.goroutines) : "—",
    dbPool,
    uptime: formatUptime(num(s.detail.uptime_seconds) ?? num(s.detail.uptime_in_seconds)),
    checked: formatRelative(s.sampled_at),
  };
}

// An hour of 30-second polls is 120 points; averaged into at most this many
// buckets the lines stay readable.
const MAX_POINTS = 30;

/**
 * One series per process for a chart. Every poll samples all services at the
 * same instant, so the poll times line the series up on one axis; consecutive
 * polls are then averaged into buckets.
 */
function seriesFor(samples: AdminServiceSample[], pick: (s: AdminServiceSample) => number | undefined) {
  const times = [...new Set(samples.map((s) => s.sampled_at))].sort();
  const perBucket = Math.max(1, Math.ceil(times.length / MAX_POINTS));
  const services = [...new Set(samples.filter((s) => pick(s) !== undefined).map((s) => s.service))].sort();
  return services.map((service, index) => {
    const byTime = new Map(samples.filter((s) => s.service === service).map((s) => [s.sampled_at, pick(s)]));
    const data = [];
    for (let i = 0; i < times.length; i += perBucket) {
      const bucket = times.slice(i, i + perBucket);
      const values = bucket.map((t) => byTime.get(t)).filter((v): v is number => v !== undefined);
      const average = values.length > 0 ? values.reduce((sum, v) => sum + v, 0) / values.length : 0;
      data.push({ date: bucket[bucket.length - 1].slice(11, 16), value: average });
    }
    return { label: serviceLabel(service), color: chartColor(index), data };
  });
}

/** Which colour is which service: the chart draws no legend of its own. */
function ChartLegend({ series }: { series: { label: string; color: string }[] }) {
  return (
    <ul className="flex flex-wrap gap-x-4 gap-y-1 px-1 pt-3 text-xs text-muted-foreground">
      {series.map((s) => (
        <li key={s.label} className="flex items-center gap-1.5">
          <span aria-hidden className="size-2.5 rounded-full" style={{ backgroundColor: s.color }} />
          {s.label}
        </li>
      ))}
    </ul>
  );
}

export default async function ServicesPage() {
  let services: AdminServices;
  let system: AdminSystem;
  try {
    const { admin } = await getApiClient();
    [services, system] = await Promise.all([admin.services(), admin.system()]);
  } catch {
    return (
      <PageShell title="Services">
        <ErrorState description="Service health could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const processes = services.series.filter((s) => s.service !== "postgres" && s.service !== "redis");
  const cpuSeries = seriesFor(processes, (s) => s.cpu_percent);
  const memorySeries = seriesFor(processes, (s) => (s.rss_bytes !== undefined ? s.rss_bytes / (1024 * 1024) : undefined));
  const down = services.latest.filter((s) => !s.up);
  const { outbox } = system;
  const stale = outbox.oldest_age_seconds > STALE_SECONDS;

  return (
    <PageShell
      title="Services"
      description="Health and resource use of every part of the platform, checked by the worker every 30 seconds. CPU is a share of the whole machine; memory is resident memory."
    >
      <div className="flex flex-col gap-6 pb-12">
        {down.length > 0 && (
          <p role="alert" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-5 py-3 text-sm text-rose-800 dark:text-rose-300">
            {down.map((s) => serviceLabel(s.service)).join(", ")} {down.length === 1 ? "is" : "are"} not answering its health check.
          </p>
        )}

        <Panel title="Services" description="The latest check of each service.">
          <div className="p-4">
            <DataTable
              columns={COLUMNS}
              rows={services.latest.map(toRow)}
              rowKey="id"
              emptyMessage="No checks yet. The worker records the first one within 30 seconds of starting; it needs MONITOR_TARGETS and MONITOR_TOKEN set."
            />
          </div>
        </Panel>

        <div className="grid gap-5 xl:grid-cols-2">
          <Panel title="CPU, last hour" description="Percent of machine capacity, per service.">
            <div className="p-4">
              {cpuSeries.length > 0 ? (
                <>
                  <AreaChart label="CPU by service, last hour" series={cpuSeries} stacked={false} formatValue={(v) => `${v.toFixed(1)}%`} />
                  <ChartLegend series={cpuSeries} />
                </>
              ) : (
                <RegisterEmpty>No CPU readings yet.</RegisterEmpty>
              )}
            </div>
          </Panel>
          <Panel title="Memory, last hour" description="Resident memory in megabytes, per service.">
            <div className="p-4">
              {memorySeries.length > 0 ? (
                <>
                  <AreaChart label="Memory by service, last hour" series={memorySeries} stacked={false} formatValue={(v) => `${Math.round(v)} MB`} />
                  <ChartLegend series={memorySeries} />
                </>
              ) : (
                <RegisterEmpty>No memory readings yet.</RegisterEmpty>
              )}
            </div>
          </Panel>
        </div>

        <Ledger title="Event queue" columns={4}>
          <LedgerCell
            label="Waiting"
            value={formatNumber(outbox.pending)}
            detail="Usage and security events not yet written"
            emptyDetail="Queue is empty"
            tone={stale ? "warning" : "neutral"}
            icon={Inbox}
          />
          <LedgerCell
            label="Retrying"
            value={formatNumber(outbox.retrying)}
            detail="Failed at least once"
            emptyDetail="Nothing failing"
            tone={outbox.retrying > 0 ? "danger" : "neutral"}
            icon={RefreshCw}
          />
          <LedgerCell
            label="Oldest waiting"
            value={outbox.oldest_pending ? formatRelative(outbox.oldest_pending) : "—"}
            detail={outbox.oldest_pending ? formatDateTime(outbox.oldest_pending) : "Queue is empty"}
            emptyDetail="Queue is empty"
            tone={stale ? "warning" : "neutral"}
            icon={Clock}
          />
          <LedgerCell
            label="Worker"
            value={stale ? "Behind" : "Keeping up"}
            detail={stale ? "Oldest event is over 5 minutes old" : "No event older than 5 minutes"}
            tone={stale ? "warning" : "success"}
            icon={AlertTriangle}
          />
        </Ledger>

        <Panel title="AI providers" description="The last probe of each upstream provider the platform routes to.">
          {system.provider_health.length > 0 ? (
            <Register>
              {system.provider_health.map((state) => (
                <RegisterRow
                  key={state.provider_kind}
                  primary={formatProviderName(state.provider_kind)}
                  secondary={[state.status_reason, state.last_error, `checked ${formatRelative(state.checked_at)}`].filter(Boolean).join(". ")}
                  value={<StatusBadge tone={healthTone(state.status)}>{state.status}</StatusBadge>}
                />
              ))}
            </Register>
          ) : (
            <RegisterEmpty>No provider has been probed yet.</RegisterEmpty>
          )}
        </Panel>
      </div>
    </PageShell>
  );
}
