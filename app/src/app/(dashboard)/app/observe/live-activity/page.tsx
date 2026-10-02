import type { Metadata } from "next";

import { AreaChart, chartColor } from "@/components/charts";
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
  formatMs,
  formatNumber,
  formatPercent,
  formatRelative,
  formatUsd,
} from "@/lib/format";
import { NOW, activityEntries, liveUsageSeries } from "@/lib/mock-data";
import { ACTIVITY_STATUS_TONE } from "@/types";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Live activity · Sentinel",
  description: "Requests streaming through the gateway right now.",
};

interface LiveRow {
  id: string;
  when: string;
  whenTs: number;
  requestId: string;
  who: string;
  project: string;
  model: string;
  provider: string;
  event: string;
  status: string;
  statusTone: StatusTone;
  tokens: number;
  latency: number;
  cost: number;
}

const liveColumns: readonly Column<LiveRow>[] = [
  { key: "when", header: "When", primary: "when", sort: "whenTs" },
  {
    key: "request",
    header: "Request",
    primary: "requestId",
    secondary: "who",
    sort: "requestId",
  },
  {
    key: "route",
    header: "Route",
    primary: "model",
    secondary: "provider",
    sort: "model",
  },
  { key: "event", header: "Event", primary: "event", sort: "event" },
  {
    key: "status",
    header: "Status",
    primary: "status",
    tone: "statusTone",
    sort: "status",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    align: "right",
    sort: "tokens",
  },
  {
    key: "latency",
    header: "Latency",
    primary: "latency",
    format: "ms",
    align: "right",
    sort: "latency",
  },
  {
    key: "cost",
    header: "Cost",
    primary: "cost",
    format: "usd",
    align: "right",
    sort: "cost",
  },
];

const recent = [...activityEntries]
  .sort((a, b) => b.occurredAt.localeCompare(a.occurredAt))
  .map((entry): LiveRow => ({
    id: entry.id,
    when: formatRelative(entry.occurredAt, NOW),
    whenTs: Date.parse(entry.occurredAt),
    requestId: entry.requestId,
    who: entry.userName,
    project: entry.projectName,
    model: entry.modelName,
    provider: entry.providerName,
    event: entry.event,
    status: entry.status,
    statusTone: ACTIVITY_STATUS_TONE[entry.status],
    tokens: entry.inputTokens + entry.outputTokens,
    latency: entry.latencyMs,
    cost: entry.cost,
  }));

const windowed = activityEntries.filter(
  (entry) => Date.parse(entry.occurredAt) >= NOW.getTime() - 3_600_000,
);
const errors = windowed.filter((entry) => entry.status === "error").length;
const blocked = windowed.filter((entry) => entry.status === "blocked").length;

export default function LiveActivityPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Live activity"
        description="Every request the gateway is processing, newest first. Streams update without a reload."
        eyebrow={
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400">
            <span className="relative flex size-2">
              <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-500 opacity-75" />
              <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
            </span>
            Streaming
          </span>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Requests this hour"
            value={formatNumber(windowed.length)}
            hint="Last 60 minutes"
            sparkline={liveUsageSeries.calls}
          />
          <StatCard
            label="Tokens this hour"
            value={formatCompact(
              windowed.reduce(
                (acc, entry) => acc + entry.inputTokens + entry.outputTokens,
                0,
              ),
            )}
            hint={`${formatCompact(liveUsageSeries.totals.inputTokens)} in / ${formatCompact(liveUsageSeries.totals.outputTokens)} out`}
            sparkline={liveUsageSeries.tokens}
          />
          <StatCard
            label="Errors"
            value={formatNumber(errors)}
            tone={errors > 0 ? "danger" : "success"}
            hint="Last 60 minutes"
          />
          <StatCard
            label="Blocked by policy"
            value={formatNumber(blocked)}
            tone={blocked > 0 ? "warning" : "success"}
            hint={`p95 ${formatMs(liveUsageSeries.totals.p95LatencyMs)}`}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Throughput by hour</CardTitle>
            <CardDescription>
              Calls and tokens across the last 24 hours, newest hour on the
              right.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <AreaChart
              label="Hourly gateway throughput"
              series={[
                { label: "Calls", color: chartColor(0), data: liveUsageSeries.calls },
                {
                  label: "Tokens / 1000",
                  color: chartColor(1),
                  data: liveUsageSeries.tokens.map((point) => ({
                    ...point,
                    value: Math.round(point.value / 1_000),
                  })),
                },
              ]}
              formatValue={formatCompact}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Request stream</CardTitle>
            <CardDescription>
              {formatNumber(activityEntries.length)} recent requests, with the
              upstream request id for support escalation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={liveColumns}
              rows={recent}
              rowKey="id"
              pageSize={12}
              emptyMessage="No requests recorded yet."
            />
          </CardContent>
        </Card>

        <p className="text-xs text-muted-foreground">
          Success rate{" "}
          {formatPercent(1 - (errors + blocked) / Math.max(1, windowed.length))}{" "}
          · Cost this hour{" "}
          {formatUsd(windowed.reduce((acc, entry) => acc + entry.cost, 0))}
        </p>
      </PageShell>
    </div>
  );
}
